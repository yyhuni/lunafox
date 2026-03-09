package application

import (
	"fmt"
	"strings"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

func buildScanTasks(workflowIDs []string, rootConfig map[string]any) ([]CreateScanTask, error) {
	var taskPlanner workflowTaskPlanner = sequentialWorkflowTaskPlanner{}
	plannedTaskItems := taskPlanner.Build(workflowIDs)
	if len(plannedTaskItems) == 0 {
		return nil, ErrCreateNoWorkflows
	}

	workflowConfigByID := make(map[string]map[string]any, len(workflowIDs))
	for _, workflowID := range workflowIDs {
		workflowConfigSlice, err := extractWorkflowConfigSlice(rootConfig, workflowID)
		if err != nil {
			return nil, err
		}
		workflowConfigByID[workflowID] = workflowConfigSlice
	}

	tasks := make([]CreateScanTask, 0, len(plannedTaskItems))
	for _, plannedItem := range plannedTaskItems {
		workflowConfigSlice, ok := workflowConfigByID[plannedItem.WorkflowID]
		if !ok {
			return nil, WrapSchemaInvalid(plannedItem.WorkflowID, "missing workflow config slice", ErrCreateInvalidConfig)
		}
		tasks = append(tasks, CreateScanTask{
			Stage:          plannedItem.Stage,
			WorkflowID:     plannedItem.WorkflowID,
			WorkflowConfig: workflowConfigSlice,
			Status:         plannedItem.InitialStatus,
		})
	}

	return tasks, nil
}

type workflowTaskPlanItem struct {
	WorkflowID    string
	Stage         int
	InitialStatus string
}

type workflowTaskPlanner interface {
	Build(workflowIDs []string) []workflowTaskPlanItem
}

type sequentialWorkflowTaskPlanner struct{}

func (sequentialWorkflowTaskPlanner) Build(workflowIDs []string) []workflowTaskPlanItem {
	out := make([]workflowTaskPlanItem, 0, len(workflowIDs))
	for index, workflowID := range workflowIDs {
		name := strings.TrimSpace(workflowID)
		if name == "" {
			continue
		}

		initialStatus := string(scandomain.TaskStatusBlocked)
		if len(out) == 0 {
			initialStatus = string(scandomain.TaskStatusPending)
		}

		out = append(out, workflowTaskPlanItem{
			WorkflowID:    name,
			Stage:         index + 1,
			InitialStatus: initialStatus,
		})
	}
	return out
}

func extractWorkflowConfigSlice(rootConfig map[string]any, workflowID string) (map[string]any, error) {
	if rootConfig == nil {
		return nil, WrapSchemaInvalid(workflowID, "configuration root must be object", ErrCreateInvalidConfig)
	}

	rawWorkflowConfig, ok := rootConfig[workflowID]
	if !ok {
		return nil, WrapSchemaInvalid(
			workflowID,
			fmt.Sprintf("missing %s config; expected nested configuration under key %q", workflowID, workflowID),
			ErrCreateInvalidConfig,
		)
	}

	workflowConfigSlice, ok := rawWorkflowConfig.(map[string]any)
	if !ok {
		return nil, WrapSchemaInvalid(
			workflowID,
			fmt.Sprintf("workflow %s configuration must be object", workflowID),
			ErrCreateInvalidConfig,
		)
	}

	return workflowConfigSlice, nil
}
