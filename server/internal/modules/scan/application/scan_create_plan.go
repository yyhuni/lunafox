package application

import (
	"errors"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

func buildScanTasks(engineNames []string, root map[string]any) ([]CreateScanTask, error) {
	enabled := scandomain.CollectEnabledWorkflowSet(engineNames, root)
	planned, err := scandomain.BuildWorkflowTaskPlan(enabled)
	if err != nil {
		if errors.Is(err, scandomain.ErrNoEnabledWorkflows) {
			return nil, ErrCreateNoWorkflows
		}
		return nil, err
	}

	tasks := make([]CreateScanTask, 0, len(planned))
	for _, item := range planned {
		tasks = append(tasks, CreateScanTask{
			Stage:        item.Stage,
			WorkflowName: string(item.Workflow),
			Status:       string(item.InitialStatus),
		})
	}
	return tasks, nil
}
