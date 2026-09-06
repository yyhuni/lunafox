package agentdata

import (
	"context"
	"errors"
	"strings"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

var (
	errTaskProgressLogNotRunning        = errors.New("task progress log task is not running")
	errTaskProgressLogOwnershipMismatch = errors.New("task progress log task is not owned by the authenticated Agent")
	errTaskProgressLogSessionMismatch   = errors.New("task progress log task session is not current")
)

type taskProgressLogApplicationDataPlane struct {
	logs  scanapp.TaskProgressLogApplicationService
	tasks scanrepo.ScanTaskRepository
}

func NewTaskProgressLogDataPlane(logs scanapp.TaskProgressLogApplicationService, tasks scanrepo.ScanTaskRepository) TaskProgressLogDataPlane {
	return &taskProgressLogApplicationDataPlane{logs: logs, tasks: tasks}
}

func (plane *taskProgressLogApplicationDataPlane) WriteTaskProgressLogs(ctx context.Context, batch TaskProgressLogBatch) (int, int, error) {
	if plane == nil || plane.logs == nil || plane.tasks == nil {
		return 0, 0, errDataPlaneDependencyMissing()
	}
	if batch.AgentID <= 0 || batch.SessionID == "" || batch.SessionID != strings.TrimSpace(batch.SessionID) || batch.SessionEpoch <= 0 {
		return 0, 0, errTaskProgressLogSessionMismatch
	}
	task, err := plane.tasks.GetByID(ctx, batch.TaskID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, 0, scanapp.ErrScanTaskNotFound
		}
		return 0, 0, err
	}
	if task == nil {
		return 0, 0, scanapp.ErrScanTaskNotFound
	}
	if task.ScanID != batch.ScanID {
		return 0, 0, scanapp.ErrScanTaskNotOwned
	}
	if task.Status != string(scandomain.TaskStatusRunning) {
		return 0, 0, errTaskProgressLogNotRunning
	}
	if task.AssignedSessionID == nil || strings.TrimSpace(*task.AssignedSessionID) == "" || *task.AssignedSessionID != batch.SessionID || task.AssignedSessionEpoch == nil || *task.AssignedSessionEpoch != batch.SessionEpoch {
		return 0, 0, errTaskProgressLogSessionMismatch
	}
	if task.AssignedAgentID == nil || *task.AssignedAgentID != batch.AgentID {
		return 0, 0, errTaskProgressLogOwnershipMismatch
	}

	entries := make([]scanapp.TaskProgressLogCreateItem, 0, len(batch.Entries))
	for _, item := range batch.Entries {
		entries = append(entries, scanapp.TaskProgressLogCreateItem{
			Sequence:  item.Sequence,
			Level:     item.Level,
			Content:   item.Content,
			EmittedAt: item.EmittedAt,
		})
	}
	return plane.logs.BatchCreateTaskProgressLogs(ctx, &scanapp.TaskProgressLogBatchCreateRequest{
		ScanID:    batch.ScanID,
		TaskID:    batch.TaskID,
		RequestID: batch.RequestID,
		Items:     entries,
	})
}

func errDataPlaneDependencyMissing() error {
	return scanapp.ErrScanTaskNotFound
}
