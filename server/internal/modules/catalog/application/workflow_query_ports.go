package application

import "context"

// WorkflowQueryStore provides read access to workflow capability metadata.
type WorkflowQueryStore interface {
	ListWorkflows(ctx context.Context) ([]Workflow, error)
	GetWorkflowByName(ctx context.Context, name string) (*Workflow, error)
}
