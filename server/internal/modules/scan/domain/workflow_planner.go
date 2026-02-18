package domain

import "fmt"

type workflowStageMode string

const (
	workflowStageSequential workflowStageMode = "sequential"
	workflowStageParallel   workflowStageMode = "parallel"
)

type workflowStage struct {
	Mode  workflowStageMode
	Flows []WorkflowName
}

type WorkflowTaskPlan struct {
	Workflow      WorkflowName
	Stage         int
	InitialStatus TaskStatus
}

var defaultWorkflowStages = []workflowStage{
	{Mode: workflowStageSequential, Flows: []WorkflowName{WorkflowSubdomainDiscovery, WorkflowPortScan, WorkflowSiteScan, WorkflowFingerprintDetect}},
	{Mode: workflowStageParallel, Flows: []WorkflowName{WorkflowURLFetch, WorkflowDirectoryScan}},
	{Mode: workflowStageSequential, Flows: []WorkflowName{WorkflowScreenshot}},
	{Mode: workflowStageSequential, Flows: []WorkflowName{WorkflowVulnScan}},
}

func CollectEnabledWorkflowSet(engineNames []string, config map[string]any) map[WorkflowName]struct{} {
	enabled := make(map[WorkflowName]struct{})
	for _, name := range engineNames {
		workflow, ok := ParseWorkflowName(name)
		if !ok {
			continue
		}
		enabled[workflow] = struct{}{}
	}
	if config == nil {
		return enabled
	}
	for key := range config {
		workflow, ok := ParseWorkflowName(key)
		if !ok {
			continue
		}
		enabled[workflow] = struct{}{}
	}
	return enabled
}

func BuildWorkflowTaskPlan(enabled map[WorkflowName]struct{}) ([]WorkflowTaskPlan, error) {
	type plannedTask struct {
		workflow WorkflowName
		stage    int
	}

	planned := make([]plannedTask, 0)
	stageIndex := 1
	for _, stage := range defaultWorkflowStages {
		flows := make([]WorkflowName, 0, len(stage.Flows))
		for _, flow := range stage.Flows {
			if _, ok := enabled[flow]; ok {
				flows = append(flows, flow)
			}
		}
		if len(flows) == 0 {
			continue
		}

		switch stage.Mode {
		case workflowStageParallel:
			for _, flow := range flows {
				planned = append(planned, plannedTask{workflow: flow, stage: stageIndex})
			}
			stageIndex++
		case workflowStageSequential:
			for _, flow := range flows {
				planned = append(planned, plannedTask{workflow: flow, stage: stageIndex})
				stageIndex++
			}
		default:
			return nil, fmt.Errorf("unsupported workflow stage mode: %s", stage.Mode)
		}
	}

	if len(planned) == 0 {
		return nil, ErrNoEnabledWorkflows
	}

	minStage := planned[0].stage
	for _, item := range planned {
		if item.stage < minStage {
			minStage = item.stage
		}
	}

	result := make([]WorkflowTaskPlan, 0, len(planned))
	for _, item := range planned {
		status := TaskStatusBlocked
		if item.stage == minStage {
			status = TaskStatusPending
		}
		result = append(result, WorkflowTaskPlan{
			Workflow:      item.workflow,
			Stage:         item.stage,
			InitialStatus: status,
		})
	}

	return result, nil
}
