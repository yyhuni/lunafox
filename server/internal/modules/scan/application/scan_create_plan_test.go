package application

import "testing"

func TestBuildScanTasks_IgnoresRootExtraWorkflowKeys(t *testing.T) {
	root := map[string]any{
		"subdomain_discovery": map[string]any{
			"apiVersion":    "v1",
			"schemaVersion": "1.0.0",
		},
		"url_fetch": map[string]any{
			"apiVersion":    "v9",
			"schemaVersion": "9.9.9",
		},
	}

	tasks, err := buildScanTasks([]string{"subdomain_discovery"}, root)
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one planned task, got %d", len(tasks))
	}
	if tasks[0].WorkflowName != "subdomain_discovery" {
		t.Fatalf("unexpected workflow planned: %s", tasks[0].WorkflowName)
	}
	if tasks[0].WorkflowAPIVersion != "v1" || tasks[0].WorkflowSchemaVersion != "1.0.0" {
		t.Fatalf("expected precomputed workflow tuple, got api=%s schema=%s", tasks[0].WorkflowAPIVersion, tasks[0].WorkflowSchemaVersion)
	}
	if tasks[0].Config == "" {
		t.Fatalf("expected task-level workflow config slice to be persisted")
	}
}

func TestBuildScanTasks_RejectsNonObjectEngineConfig(t *testing.T) {
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
			"apiVersion":    "v1",
			"schemaVersion": "1.0.0",
			"enabled":       true,
		},
	}

	tasks, err := buildScanTasks([]string{"future_workflow"}, root)
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one planned task, got %d", len(tasks))
	}
	if tasks[0].WorkflowName != "future_workflow" {
		t.Fatalf("unexpected workflow planned: %s", tasks[0].WorkflowName)
	}
}
