package application

import (
	"context"
	"errors"

	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

var (
	ErrScanTaskNotFound          = errors.New("scan task not found")
	ErrScanTaskNotOwned          = errors.New("scan task not owned by agent")
	ErrScanTaskInvalidTransition = errors.New("invalid scan task transition")
	ErrScanTaskInvalidUpdate     = errors.New("invalid scan task update")
	ErrScanNoWorkflows           = errors.New("no workflows enabled for scan")
)

type ScanTaskFacade struct{ runtimeService *TaskRuntimeService }

func NewScanTaskFacade(taskStore TaskStore, runtimeStore TaskRuntimeScanStore) *ScanTaskFacade {
	return &ScanTaskFacade{runtimeService: NewTaskRuntimeService(taskStore, runtimeStore)}
}

func (service *ScanTaskFacade) PullTask(ctx context.Context, agentID int) (*TaskAssignment, error) {
	taskAssignment, err := service.runtimeService.PullTask(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if taskAssignment == nil {
		return nil, nil
	}

	pkg.Debug("Task assigned to agent", zap.Int("task.id", taskAssignment.TaskID), zap.Int("agent.id", agentID), zap.Int("scan.id", taskAssignment.ScanID), zap.Int("stage", taskAssignment.Stage), zap.String("workflow.id", taskAssignment.WorkflowID))

	return taskAssignment, nil
}
