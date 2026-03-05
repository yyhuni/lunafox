package application

import (
	"context"
	"strings"
)

// WorkflowCatalogFacade provides read-only workflow catalog queries.
type WorkflowCatalogFacade struct {
	workflowStore        WorkflowQueryStore
	workflowProfileStore WorkflowProfileQueryStore
}

// NewWorkflowCatalogFacade creates a workflow catalog facade.
func NewWorkflowCatalogFacade(workflowStore WorkflowQueryStore, workflowProfileStore WorkflowProfileQueryStore) *WorkflowCatalogFacade {
	return &WorkflowCatalogFacade{
		workflowStore:        workflowStore,
		workflowProfileStore: workflowProfileStore,
	}
}

// ListWorkflows returns all available workflows.
func (service *WorkflowCatalogFacade) ListWorkflows() ([]Workflow, error) {
	return service.workflowStore.ListWorkflows(context.Background())
}

// GetWorkflowByID returns a workflow by id.
func (service *WorkflowCatalogFacade) GetWorkflowByID(workflowID string) (*Workflow, error) {
	trimmed := strings.TrimSpace(workflowID)
	if trimmed == "" {
		return nil, ErrWorkflowNotFound
	}
	return service.workflowStore.GetWorkflowByID(context.Background(), trimmed)
}

// ListProfiles returns all workflow profiles.
func (service *WorkflowCatalogFacade) ListProfiles() ([]WorkflowProfile, error) {
	return service.workflowProfileStore.ListWorkflowProfiles(context.Background())
}

// GetProfileByID returns a workflow profile by ID.
func (service *WorkflowCatalogFacade) GetProfileByID(id string) (*WorkflowProfile, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return nil, ErrWorkflowProfileNotFound
	}
	return service.workflowProfileStore.GetWorkflowProfileByID(context.Background(), trimmed)
}
