package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type agentLocationMapService interface {
	Current(ctx context.Context) (agentapp.AgentLocationMap, error)
}

type AgentLocationMapHandler struct {
	service agentLocationMapService
}

func NewAgentLocationMapHandler(service agentLocationMapService) *AgentLocationMapHandler {
	return &AgentLocationMapHandler{service: service}
}

// Current returns the complete positioned-Agent map fixed view.
// GET /v1/admin/agentLocationMaps/current
func (handler *AgentLocationMapHandler) Current(c *gin.Context) {
	if handler == nil || handler.service == nil {
		httpdto.InternalError(c, "Agent location map service is not configured")
		return
	}
	locationMap, err := handler.service.Current(c.Request.Context())
	if err != nil {
		httpdto.InternalError(c, "Failed to get Agent location map")
		return
	}
	httpdto.Success(c, toAgentLocationMapOutput(locationMap))
}

func toAgentLocationMapOutput(locationMap agentapp.AgentLocationMap) dto.AgentLocationMapResponse {
	agents := make([]dto.AgentLocationMapAgentResponse, 0, len(locationMap.Agents))
	for _, agent := range locationMap.Agents {
		agents = append(agents, dto.AgentLocationMapAgentResponse{
			Name:          httpdto.AgentName(agent.AgentID),
			DisplayName:   agent.DisplayName,
			Status:        agent.Status,
			HealthState:   agent.HealthState,
			TaskSlotsUsed: copyAgentLocationMapTaskSlotsOutput(agent.TaskSlotsUsed),
			Location: dto.AgentLocationMapLocationResponse{
				State:            string(agent.Location.State),
				Latitude:         agent.Location.Latitude,
				Longitude:        agent.Location.Longitude,
				AccuracyRadiusKM: copyAgentLocationMapRadiusOutput(agent.Location.AccuracyRadiusKM),
				SourceObservedIP: agent.Location.SourceObservedIP,
				ProviderKey:      agent.Location.ProviderKey,
				ResolvedAt:       agent.Location.ResolvedAt.UTC(),
			},
		})
	}
	return dto.AgentLocationMapResponse{
		Name:           httpdto.AgentLocationMapName(),
		GeneratedAt:    locationMap.GeneratedAt.UTC(),
		ServerLocation: toServerLocationMapOutput(locationMap.ServerLocation),
		Agents:         agents,
	}
}

func toServerLocationMapOutput(location *agentapp.ServerLocationRead) *dto.ServerLocationMapResponse {
	if location == nil {
		return nil
	}
	return &dto.ServerLocationMapResponse{
		State:            string(location.State),
		ObservedEgressIP: location.ObservedEgressIP,
		Latitude:         location.Latitude,
		Longitude:        location.Longitude,
		AccuracyRadiusKM: copyAgentLocationMapRadiusOutput(location.AccuracyRadiusKM),
		ProviderKey:      location.ProviderKey,
		ResolvedAt:       location.ResolvedAt.UTC(),
	}
}

func copyAgentLocationMapTaskSlotsOutput(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func copyAgentLocationMapRadiusOutput(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
