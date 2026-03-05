package dto

import catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"

type WorkflowResponse struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

func NewWorkflowResponse(workflow *catalogdomain.Workflow) WorkflowResponse {
	return WorkflowResponse{
		Name:        workflow.Name,
		Title:       workflow.Title,
		Description: workflow.Description,
	}
}

func NewWorkflowListResponse(workflows []catalogdomain.Workflow) []WorkflowResponse {
	responses := make([]WorkflowResponse, len(workflows))
	for i := range workflows {
		responses[i] = NewWorkflowResponse(&workflows[i])
	}
	return responses
}
