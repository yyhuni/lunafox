package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type TaskRuntimeService struct {
	taskStore    TaskStore
	runtimeStore TaskRuntimeScanStore
}

func NewTaskRuntimeService(taskStore TaskStore, runtimeStore TaskRuntimeScanStore) *TaskRuntimeService {
	return &TaskRuntimeService{taskStore: taskStore, runtimeStore: runtimeStore}
}

func (service *TaskRuntimeService) PullTask(ctx context.Context, agentID int) (*TaskAssignment, error) {
	task, err := service.taskStore.PullTask(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, nil
	}

	scan, err := service.runtimeStore.GetTaskRuntimeByID(task.ScanID)
	if err != nil {
		return nil, err
	}

	scanStatus, ok := scandomain.ParseScanStatus(scan.Status)
	if !ok {
		return nil, ErrTaskInvalidTransition
	}
	// The first successful task pull promotes scan lifecycle from pending to running.
	if scanStatus == scandomain.ScanStatusPending {
		domainScan := &scandomain.Scan{Status: scanStatus}
		if err := domainScan.MarkRunning(); err != nil {
			return nil, err
		}
		if err := service.runtimeStore.UpdateStatus(scan.ID, string(domainScan.Status), ""); err != nil {
			return nil, err
		}
	}

	config := strings.TrimSpace(task.Config)
	// Task-level config takes precedence; fall back to scan-level YAML when empty.
	if config == "" {
		config = strings.TrimSpace(scan.YamlConfiguration)
	}
	assignment := &TaskAssignment{TaskID: task.ID, ScanID: task.ScanID, Stage: task.Stage, WorkflowName: task.WorkflowName, TargetID: scan.TargetID, WorkspaceDir: workspaceDir(task.ScanID, task.ID), Config: config}
	if scan.Target != nil {
		assignment.TargetName = scan.Target.Name
		assignment.TargetType = scan.Target.Type
	}
	return assignment, nil
}

func (service *TaskRuntimeService) UpdateStatus(ctx context.Context, agentID, taskID int, status, errorMessage string) error {
	status = strings.TrimSpace(status)
	errorMessage = strings.TrimSpace(errorMessage)
	if status == "" {
		return ErrTaskInvalidUpdate
	}

	nextStatus, ok := scandomain.ParseTaskStatus(status)
	if !ok {
		return ErrTaskInvalidTransition
	}
	if nextStatus == scandomain.TaskStatusFailed && errorMessage == "" {
		return ErrTaskInvalidUpdate
	}

	task, err := service.taskStore.GetByID(ctx, taskID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrTaskNotFound
		}
		return err
	}
	if task.AgentID == nil || *task.AgentID != agentID {
		return ErrTaskNotOwned
	}

	currentStatus, ok := scandomain.ParseTaskStatus(task.Status)
	if !ok {
		return ErrTaskInvalidTransition
	}
	if currentStatus == nextStatus {
		return nil
	}

	domainTask := &scandomain.ScanTask{Status: currentStatus}
	if err := domainTask.ApplyAgentResult(nextStatus, errorMessage, time.Now().UTC()); err != nil {
		if errors.Is(err, scandomain.ErrFailureMessageMissing) {
			return ErrTaskInvalidUpdate
		}
		return ErrTaskInvalidTransition
	}

	if err := service.taskStore.UpdateStatus(ctx, taskID, string(nextStatus), errorMessage); err != nil {
		return err
	}
	// Stage unlock and scan-level status transitions happen only after a terminal task result.
	if scandomain.IsTerminalTaskStatus(nextStatus) {
		if err := service.unlockNextStageIfReady(ctx, task.ScanID, task.Stage); err != nil {
			return err
		}
		return service.recalculateScanStatus(ctx, task.ScanID)
	}

	return nil
}

func (service *TaskRuntimeService) unlockNextStageIfReady(ctx context.Context, scanID, stage int) error {
	// Keep stage ordering strict: do not release next stage while current one is still active.
	active, err := service.taskStore.CountActiveByScanAndStage(ctx, scanID, stage)
	if err != nil {
		return err
	}
	if active > 0 {
		return nil
	}

	_, err = service.taskStore.UnlockNextStage(ctx, scanID, stage)
	return err
}

func (service *TaskRuntimeService) recalculateScanStatus(
	ctx context.Context,
	scanID int,
) error {
	pending, running, _, failed, cancelled, err := service.taskStore.GetStatusCountsByScanID(ctx, scanID)
	if err != nil {
		return err
	}

	nextStatus, shouldUpdate := scandomain.ResolveScanStatusFromTaskCounts(pending, running, failed, cancelled)
	if !shouldUpdate {
		return nil
	}
	if nextStatus == scandomain.ScanStatusFailed && strings.TrimSpace(lastErrorMessage) != "" {
		return service.runtimeStore.UpdateStatus(scanID, string(nextStatus), lastErrorMessage)
	}
	return service.runtimeStore.UpdateStatus(scanID, string(nextStatus), "")
}

func workspaceDir(scanID, taskID int) string {
	// Shared-data contract: /opt/lunafox is mounted volume across server/agent/worker.
	return fmt.Sprintf("/opt/lunafox/results/scan_%d/task_%d", scanID, taskID)
}
