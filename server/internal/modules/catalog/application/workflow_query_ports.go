package application

import "context"

// WorkflowQueryStore provides read access to workflow capability metadata.
type WorkflowQueryStore interface {
	ListWorkflows(ctx context.Context) ([]Workflow, error)
	GetWorkflowByID(ctx context.Context, workflowID string) (*Workflow, error)
}
