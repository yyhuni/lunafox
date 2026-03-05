package dto

import catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"

type ProfileResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	WorkflowIDs   []string `json:"workflowIds"`
	Configuration string   `json:"configuration"`
}

func NewProfileResponse(p *catalogdomain.WorkflowProfile) ProfileResponse {
	workflowIDs := append([]string(nil), p.WorkflowIDs...)
	return ProfileResponse{
		ID:            p.ID,
		Name:          p.Name,
		Description:   p.Description,
		WorkflowIDs:   workflowIDs,
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
