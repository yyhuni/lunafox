package domain

import (
	"errors"
	"testing"
)

func TestCollectEnabledWorkflowSet(t *testing.T) {
	enabled := CollectEnabledWorkflowSet(
		[]string{" subdomain_discovery ", "URL_FETCH", "unknown"},
		map[string]any{"directory_scan": true, "URL_FETCH": map[string]any{}, "invalid": 1},
	)

	if len(enabled) != 3 {
		t.Fatalf("expected 3 enabled workflows, got %d", len(enabled))
	}
	if _, ok := enabled[WorkflowSubdomainDiscovery]; !ok {
		t.Fatalf("expected subdomain_discovery enabled")
	}
	if _, ok := enabled[WorkflowURLFetch]; !ok {
		t.Fatalf("expected url_fetch enabled")
	}
	if _, ok := enabled[WorkflowDirectoryScan]; !ok {
		t.Fatalf("expected directory_scan enabled")
	}
}

func TestBuildWorkflowTaskPlan(t *testing.T) {
	enabled := map[WorkflowName]struct{}{
		WorkflowSubdomainDiscovery: {},
		WorkflowURLFetch:           {},
		WorkflowDirectoryScan:      {},
		WorkflowScreenshot:         {},
	}

	plan, err := BuildWorkflowTaskPlan(enabled)
	if err != nil {
		t.Fatalf("build workflow task plan failed: %v", err)
	}
	if len(plan) != 4 {
		t.Fatalf("expected 4 task plans, got %d", len(plan))
	}

	expected := []WorkflowTaskPlan{
		{Workflow: WorkflowSubdomainDiscovery, Stage: 1, InitialStatus: TaskStatusPending},
		{Workflow: WorkflowURLFetch, Stage: 2, InitialStatus: TaskStatusBlocked},
		{Workflow: WorkflowDirectoryScan, Stage: 2, InitialStatus: TaskStatusBlocked},
		{Workflow: WorkflowScreenshot, Stage: 3, InitialStatus: TaskStatusBlocked},
	}

	for index := range expected {
		if plan[index].Workflow != expected[index].Workflow {
			t.Fatalf("unexpected workflow at %d, want=%s got=%s", index, expected[index].Workflow, plan[index].Workflow)
		}
		if plan[index].Stage != expected[index].Stage {
			t.Fatalf("unexpected stage at %d, want=%d got=%d", index, expected[index].Stage, plan[index].Stage)
		}
		if plan[index].InitialStatus != expected[index].InitialStatus {
			t.Fatalf("unexpected status at %d, want=%s got=%s", index, expected[index].InitialStatus, plan[index].InitialStatus)
		}
	}
}

func TestBuildWorkflowTaskPlan_NoEnabledWorkflows(t *testing.T) {
	_, err := BuildWorkflowTaskPlan(map[WorkflowName]struct{}{})
	if !errors.Is(err, ErrNoEnabledWorkflows) {
		t.Fatalf("expected ErrNoEnabledWorkflows, got %v", err)
	}
}
