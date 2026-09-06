package handler

import (
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func toAgentOutput(agent *agentdomain.Agent, heartbeat *dto.AgentHeartbeatResponse) dto.AgentResponse {
	return dto.AgentResponse{
		ID:               agent.ID,
		Name:             httpdto.AgentName(agent.ID),
		InstanceID:       agent.InstanceID,
		DisplayName:      agent.DisplayName,
		Status:           agent.Status,
		ObservedHostname: agent.ObservedHostname,
		ConnectionIP:     agent.ConnectionIP,
		AgentVersion:     agent.AgentVersion,
		MaxTasks:         agent.MaxTasks,
		CPUThreshold:     agent.CPUThreshold,
		MemThreshold:     agent.MemThreshold,
		DiskThreshold:    agent.DiskThreshold,
		ConnectedAt:      timeutil.ToUTCPtr(agent.ConnectedAt),
		LastHeartbeat:    timeutil.ToUTCPtr(agent.LastHeartbeat),
		Health: dto.HealthStatus{
			State:   agent.HealthState,
			Reason:  agent.HealthReason,
			Message: agent.HealthMessage,
			Since:   agent.HealthSince,
		},
		Heartbeat: heartbeat,
		CreatedAt: timeutil.ToUTC(agent.CreatedAt),
	}
}

func toAgentDetailOutput(agent *agentdomain.Agent, heartbeat *dto.AgentHeartbeatResponse, at time.Time) dto.AgentDetailResponse {
	response := dto.AgentDetailResponse{
		AgentResponse:        toAgentOutput(agent, heartbeat),
		ObservedSourceIP:     agent.ObservedSourceIP,
		ObservedIPGeneration: agent.ObservedIPGeneration,
		LocationState:        string(agent.Location.StateAt(at)),
	}
	if agent.Location == nil {
		return response
	}
	location := agent.Location
	response.Location = &dto.AgentLocationResponse{
		Latitude:         location.Latitude,
		Longitude:        location.Longitude,
		AccuracyRadiusKM: copyAgentDetailRadius(location.AccuracyRadiusKM),
		SourceObservedIP: location.SourceObservedIP,
		ProviderKey:      location.ProviderKey,
		ResolvedAt:       location.ResolvedAt.UTC(),
	}
	return response
}

func copyAgentDetailRadius(radius *float64) *float64 {
	if radius == nil {
		return nil
	}
	copy := *radius
	return &copy
}
