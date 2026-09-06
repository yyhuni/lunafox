package application

import "testing"

func TestBuildPlanTaskScanTasksCreatesOnePendingSkeletonPerWorkflowStep(t *testing.T) {
	manifest := ScanCreateWorkflowManifest{
		ScanWorkflowID: "default",
		Stages: []ScanCreateWorkflowStage{
			{StageID: "discovery", Steps: []ScanCreateWorkflowStep{{StepID: "discover", EngineID: "engine.lunafox.subdomain_discovery"}}},
			{StageID: "ports", Steps: []ScanCreateWorkflowStep{{StepID: "scan", EngineID: "engine.lunafox.port_scan"}}},
		},
	}

	tasks, err := buildPlanTaskScanTasks(manifest)
	if err != nil {
		t.Fatalf("buildPlanTaskScanTasks() error = %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("task count = %d, want 2", len(tasks))
	}
	if tasks[0].StageOrder != 1 || tasks[0].StageID != "discovery" || tasks[0].StepOrder != 1 || tasks[0].Status != CreateTaskStatusPending {
		t.Fatalf("unexpected first task skeleton: %+v", tasks[0])
	}
	if tasks[1].StageOrder != 2 || tasks[1].StageID != "ports" || tasks[1].StepOrder != 1 || tasks[1].Status != CreateTaskStatusPending {
		t.Fatalf("unexpected second task skeleton: %+v", tasks[1])
	}
	for index, task := range tasks {
		if task.ResolvedExecutionPlan != nil || task.SkipReason != "" {
			t.Fatalf("task %d carried finalized state before PlanTask: %+v", index, task)
		}
	}
}

func TestBuildPlanTaskScanTasksRejectsEmptyWorkflow(t *testing.T) {
	_, err := buildPlanTaskScanTasks(ScanCreateWorkflowManifest{ScanWorkflowID: "empty"})
	if err != ErrCreateNoScanWorkflows {
		t.Fatalf("buildPlanTaskScanTasks() error = %v, want %v", err, ErrCreateNoScanWorkflows)
	}
}
