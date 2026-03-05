package dto

import catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"

type ProfileResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	WorkflowNames []string `json:"workflowNames"`
	Configuration string   `json:"configuration"`
}

func NewProfileResponse(p *catalogdomain.WorkflowProfile) ProfileResponse {
	workflowNames := append([]string(nil), p.WorkflowNames...)
	return ProfileResponse{
		ID:            p.ID,
		Name:          p.Name,
		Description:   p.Description,
		WorkflowNames: workflowNames,
		Configuration: p.Configuration,
	}
}

func NewProfileListResponse(profiles []catalogdomain.WorkflowProfile) []ProfileResponse {
	responses := make([]ProfileResponse, len(profiles))
	for i := range profiles {
		responses[i] = NewProfileResponse(&profiles[i])
	}
	return responses
}
