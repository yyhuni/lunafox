package dto

import catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"

type WorkflowResponse struct {
	WorkflowID  string                   `json:"workflowId"`
	DisplayName string                   `json:"displayName,omitempty"`
	Description string                   `json:"description,omitempty"`
	Executor    WorkflowExecutorResponse `json:"executor"`
}

type WorkflowExecutorResponse struct {
	Type string `json:"type"`
	Ref  string `json:"ref"`
}

func NewWorkflowResponse(workflow *catalogdomain.Workflow) WorkflowResponse {
	return WorkflowResponse{
		WorkflowID:  workflow.WorkflowID,
		DisplayName: workflow.DisplayName,
		Description: workflow.Description,
		Executor: WorkflowExecutorResponse{
			Type: workflow.Executor.Type,
			Ref:  workflow.Executor.Ref,
		},
	}
}

func NewWorkflowListResponse(workflows []catalogdomain.Workflow) []WorkflowResponse {
	responses := make([]WorkflowResponse, len(workflows))
	for i := range workflows {
		responses[i] = NewWorkflowResponse(&workflows[i])
	}
	return responses
}
