package dto

import catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"

type ProfileResponse struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	WorkflowIDs   []string       `json:"workflowIds"`
	Configuration map[string]any `json:"configuration"`
}

func NewProfileResponse(profile *catalogdomain.WorkflowProfile) ProfileResponse {
	workflowIDs := append([]string(nil), profile.WorkflowIDs...)
	configuration := make(map[string]any, len(profile.Configuration))
	for key, value := range profile.Configuration {
		configuration[key] = value
	}

	return ProfileResponse{
		ID:            profile.ID,
		Name:          profile.Name,
		Description:   profile.Description,
		WorkflowIDs:   workflowIDs,
		Configuration: configuration,
	}
}

func NewProfileListResponse(profiles []catalogdomain.WorkflowProfile) []ProfileResponse {
	responses := make([]ProfileResponse, len(profiles))
	for index := range profiles {
		responses[index] = NewProfileResponse(&profiles[index])
	}
	return responses
}
