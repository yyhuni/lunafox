package dto

import "github.com/yyhuni/lunafox/server/internal/preset"

type PresetResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Configuration string `json:"configuration"`
}

func NewPresetResponse(p *preset.Preset) PresetResponse {
	return PresetResponse{
		ID:            p.ID,
		Name:          p.Name,
		Description:   p.Description,
		Configuration: p.Configuration,
	}
}

func NewPresetListResponse(presets []preset.Preset) []PresetResponse {
	responses := make([]PresetResponse, len(presets))
	for i := range presets {
		responses[i] = NewPresetResponse(&presets[i])
	}
	return responses
}
