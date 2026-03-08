package application

import "testing"

func TestBuildScanTasks_IgnoresRootExtraWorkflowKeys(t *testing.T) {
	root := map[string]any{
		"subdomain_discovery": map[string]any{
			"enabled": true,
		},
		"url_fetch": map[string]any{
			"enabled": true,
		},
	}

	tasks, err := buildScanTasks([]string{"subdomain_discovery"}, root)
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one planned task, got %d", len(tasks))
	}
	if tasks[0].WorkflowID != "subdomain_discovery" {
		t.Fatalf("unexpected workflow planned: %s", tasks[0].WorkflowID)
	}
	if tasks[0].WorkflowConfig == nil {
		t.Fatalf("expected task-level workflow config object to be persisted")
	}
}

func TestBuildScanTasks_RejectsNonObjectWorkflowConfig(t *testing.T) {
	root := map[string]any{
		"subdomain_discovery": "invalid",
	}

	_, err := buildScanTasks([]string{"subdomain_discovery"}, root)
	if err == nil {
		t.Fatalf("expected schema error")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok {
		t.Fatalf("expected workflow error, got: %v", err)
	}
	if workflowErr.Code != WorkflowErrorCodeSchemaInvalid {
		t.Fatalf("unexpected code: %s", workflowErr.Code)
	}
}

func TestBuildScanTasks_DoesNotDependOnDomainWorkflowSwitch(t *testing.T) {
	root := map[string]any{
		"future_workflow": map[string]any{
			"enabled": true,
		},
	}

	tasks, err := buildScanTasks([]string{"future_workflow"}, root)
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one planned task, got %d", len(tasks))
	}
	if tasks[0].WorkflowID != "future_workflow" {
		t.Fatalf("unexpected workflow planned: %s", tasks[0].WorkflowID)
	}
	if tasks[0].WorkflowConfig == nil {
		t.Fatalf("expected workflow config object on planned task")
	}
}
