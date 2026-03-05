package catalogwiring

import (
	"context"
	"testing"

	workflowschema "github.com/yyhuni/lunafox/server/internal/workflow/schema"
)

func TestCatalogWorkflowQueryStoreAdapterListWorkflowsSortsByWorkflowID(t *testing.T) {
	adapter := &catalogWorkflowQueryStoreAdapter{
		listMetadata: func() ([]workflowschema.WorkflowMetadata, error) {
			return []workflowschema.WorkflowMetadata{
				{WorkflowID: "z_workflow", DisplayName: "Z"},
				{WorkflowID: "a_workflow", DisplayName: "A"},
			}, nil
		},
	}

	workflows, err := adapter.ListWorkflows(context.Background())
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}
	if len(workflows) != 2 {
		t.Fatalf("expected 2 workflows, got %d", len(workflows))
	}
	if workflows[0].WorkflowID != "a_workflow" || workflows[1].WorkflowID != "z_workflow" {
		t.Fatalf("expected workflowId ASC order, got %+v", workflows)
	}
}

