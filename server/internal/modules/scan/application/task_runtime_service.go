package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/contracts/runtimecontract"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
	"go.uber.org/zap"
)

type TaskRuntimeService struct {
	taskStore    TaskStore
	runtimeStore TaskRuntimeScanStore
}

func NewTaskRuntimeService(
	taskStore TaskStore,
	runtimeStore TaskRuntimeScanStore,
) *TaskRuntimeService {
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

	workflowConfig := task.WorkflowConfig
	if len(workflowConfig) == 0 {
		configErr := NewWorkflowError(
			WorkflowErrorCodeWorkflowConfigInvalid,
			WorkflowErrorStageServerSchemaGate,
			"workflowConfig",
			"task-level workflow config object is missing",
			nil,
		)
		return nil, service.handlePulledTaskRejection(ctx, task, scan, agentID, configErr)
	}

	workflowID := strings.TrimSpace(task.WorkflowID)
	if workflowID == "" {
		configErr := WrapSchemaInvalid("workflow", "task workflowId is missing", nil)
		return nil, service.handlePulledTaskRejection(ctx, task, scan, agentID, configErr)
	}

	if scanStatus == scandomain.ScanStatusPending {
		domainScan := &scandomain.Scan{Status: scanStatus}
		if err := domainScan.MarkRunning(); err != nil {
			return nil, err
		}
		if err := service.runtimeStore.UpdateStatus(scan.ID, string(domainScan.Status), nil); err != nil {
			return nil, err
		}
	}

	taskAssignment := &TaskAssignment{
		TaskID:         task.ID,
		ScanID:         task.ScanID,
		Stage:          task.Stage,
		WorkflowID:     task.WorkflowID,
		TargetID:       scan.TargetID,
		WorkspaceDir:   workspaceDir(task.ScanID, task.ID),
		WorkflowConfig: workflowConfig,
	}
	if scan.Target != nil {
		taskAssignment.TargetName = scan.Target.Name
		taskAssignment.TargetType = scan.Target.Type
	}

	return taskAssignment, nil
}

func (service *TaskRuntimeService) UpdateStatus(
	ctx context.Context,
	agentID int,
	taskID int,
	status string,
	failure *FailureDetail,
) error {
	status = strings.TrimSpace(status)
	if status == "" {
		return ErrTaskInvalidUpdate
	}

	nextStatus, ok := scandomain.ParseTaskStatus(status)
	if !ok {
		return ErrTaskInvalidTransition
	}

	normalizedFailure, err := normalizeTaskFailure(nextStatus, failure)
	if err != nil {
		return err
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
	if err := domainTask.ApplyAgentResult(nextStatus, failureMessage(normalizedFailure), time.Now().UTC()); err != nil {
		if errors.Is(err, scandomain.ErrFailureMessageMissing) {
			return ErrTaskInvalidUpdate
		}
		return ErrTaskInvalidTransition
	}

	if err := service.taskStore.UpdateStatus(ctx, taskID, string(nextStatus), normalizedFailure); err != nil {
		return err
	}
	if scandomain.IsTerminalTaskStatus(nextStatus) {
		if err := service.unlockNextStageIfReady(ctx, task.ScanID, task.Stage); err != nil {
			return err
		}
		return service.recalculateScanStatus(ctx, task.ScanID)
	}

	return nil
}

func (service *TaskRuntimeService) unlockNextStageIfReady(ctx context.Context, scanID, stage int) error {
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

	nextStatus, shouldUpdate := scandomain.ResolveScanStatusFromTaskCounts(
		pending,
		running,
		failed,
		cancelled,
	)
	if !shouldUpdate {
		return nil
	}
	if nextStatus == scandomain.ScanStatusFailed {
		failedTasks, err := service.taskStore.ListFailedByScanID(ctx, scanID)
		if err != nil {
			return err
		}
		canonicalTask, err := selectCanonicalFailedTask(failedTasks)
		if err != nil {
			return err
		}
		return service.runtimeStore.UpdateStatus(scanID, string(nextStatus), cloneFailureDetail(canonicalTask.Failure))
	}

	return service.runtimeStore.UpdateStatus(scanID, string(nextStatus), nil)
}

func workspaceDir(scanID, taskID int) string {
	return runtimecontract.BuildTaskWorkspaceDir(scanID, taskID)
}

func (service *TaskRuntimeService) failPulledTask(
	ctx context.Context,
	task *TaskRecord,
	scan *TaskScanRecord,
	rejectReason string,
	originalErr error,
) error {
	if task == nil {
		return originalErr
	}

	failure := failureFromReasonAndError(rejectReason, originalErr)
	if err := service.taskStore.FailPulledTask(ctx, task.ID, failure); err != nil {
		pkg.Error(
			"Fail pulled task after scheduler rejection failed",
			zap.Int("task.id", task.ID),
			zap.Int("scan.id", scanIDOf(scan)),
			zap.String("reason", rejectReason),
			zap.Error(err),
		)
		return err
	}
	if scan != nil {
		if recalcErr := service.recalculateScanStatus(ctx, scan.ID); recalcErr != nil {
			pkg.Error(
				"Recalculate scan status after pulled task failure failed",
				zap.Int("scan.id", scan.ID),
				zap.Error(recalcErr),
			)
			return recalcErr
		}
	}

	return originalErr
}

func (service *TaskRuntimeService) handlePulledTaskRejection(
	ctx context.Context,
	task *TaskRecord,
	scan *TaskScanRecord,
	agentID int,
	err error,
) error {
	reason := rejectionFailureKindForWorkflowError(err)
	pkg.Warn(
		"Rejecting pulled task during task assignment",
		zap.Int("task.id", taskIDOf(task)),
		zap.Int("scan.id", scanIDOf(scan)),
		zap.Int("agent.id", agentID),
		zap.String("reason", reason),
		zap.Error(err),
	)
	return service.failPulledTask(ctx, task, scan, reason, err)
}

func taskIDOf(task *TaskRecord) int {
	if task == nil {
		return 0
	}
	return task.ID
}

func scanIDOf(scan *TaskScanRecord) int {
	if scan == nil {
		return 0
	}
	return scan.ID
}

func rejectionFailureKindForWorkflowError(err error) string {
	if workflowErr, ok := AsWorkflowError(err); ok {
		return workflowErrorCodeToFailureKind(workflowErr.Code)
	}
	return "scheduler_rejected"
}

func errorMessageOf(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}

func normalizeTaskFailure(status scandomain.TaskStatus, failure *FailureDetail) (*FailureDetail, error) {
	if status != scandomain.TaskStatusFailed {
		if failure != nil && (strings.TrimSpace(failure.Kind) != "" || strings.TrimSpace(failure.Message) != "") {
			return nil, ErrTaskInvalidUpdate
		}
		return nil, nil
	}
	if failure == nil {
		return nil, ErrTaskInvalidUpdate
	}
	message := strings.TrimSpace(failure.Message)
	if message == "" {
		return nil, ErrTaskInvalidUpdate
	}
	kind := canonicalFailureKind(strings.TrimSpace(failure.Kind))
	if kind == "" {
		kind = "unknown"
	}
	return &FailureDetail{Kind: kind, Message: message}, nil
}

func failureFromReasonAndError(reason string, originalErr error) *FailureDetail {
	message := strings.TrimSpace(errorMessageOf(originalErr))
	if message == "" {
		message = "scheduler rejected task as non-recoverable"
	}
	kind := canonicalFailureKind(strings.TrimSpace(reason))
	if kind == "" {
		kind = "unknown"
	}
	return &FailureDetail{Kind: kind, Message: message}
}

func failureMessage(failure *FailureDetail) string {
	if failure == nil {
		return ""
	}
	return failure.Message
}

func cloneFailureDetail(failure *FailureDetail) *FailureDetail {
	if failure == nil {
		return nil
	}
	cloned := *failure
	return &cloned
}

func selectCanonicalFailedTask(tasks []TaskRecord) (*TaskRecord, error) {
	var best *TaskRecord
	for index := range tasks {
		candidate := &tasks[index]
		if candidate == nil || candidate.Failure == nil {
			continue
		}
		if strings.TrimSpace(candidate.Failure.Message) == "" {
			continue
		}
		if best == nil || canonicalTaskLess(candidate, best) {
			best = candidate
		}
	}
	if best == nil {
		return nil, fmt.Errorf("failed scan has no canonical failed task")
	}
	return best, nil
}

func canonicalTaskLess(left, right *TaskRecord) bool {
	leftPriority := failurePriority(left.Failure.Kind)
	rightPriority := failurePriority(right.Failure.Kind)
	if leftPriority != rightPriority {
		return leftPriority < rightPriority
	}
	if left.Stage != right.Stage {
		return left.Stage < right.Stage
	}
	if compared := compareTimePtr(left.CompletedAt, right.CompletedAt); compared != 0 {
		return compared < 0
	}
	return left.ID < right.ID
}

func compareTimePtr(left, right *time.Time) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return 1
	}
	if right == nil {
		return -1
	}
	if left.Before(*right) {
		return -1
	}
	if left.After(*right) {
		return 1
	}
	return 0
}

func failurePriority(kind string) int {
	switch canonicalFailureKind(kind) {
	case "schema_invalid", "workflow_config_invalid", "workflow_prereq_missing", "decode_config_failed":
		return 1
	case "scheduler_rejected":
		return 2
	case "worker_start_failed", "agent_disconnected":
		return 3
	case "task_timeout", "container_wait_failed", "container_exit_failed":
		return 4
	case "runtime_error", "unknown":
		return 5
	default:
		return 5
	}
}

func workflowErrorCodeToFailureKind(code string) string {
	switch strings.TrimSpace(code) {
	case WorkflowErrorCodeSchemaInvalid:
		return "schema_invalid"
	case WorkflowErrorCodeWorkflowConfigInvalid:
		return "workflow_config_invalid"
	case WorkflowErrorCodeWorkflowPrereqMissing:
		return "workflow_prereq_missing"
	default:
		return canonicalFailureKind(code)
	}
}

func canonicalFailureKind(value string) string {
	return strings.TrimSpace(value)
}
