package application

import "context"

// WorkflowProfileQueryStore provides read access to workflow profile templates.
type WorkflowProfileQueryStore interface {
	ListWorkflowProfiles(ctx context.Context) ([]WorkflowProfile, error)
	GetWorkflowProfileByID(ctx context.Context, id string) (*WorkflowProfile, error)
}
