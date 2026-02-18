package repository

import (
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func modelAgentToDomain(agent *model.Agent) *agentdomain.Agent {
	if agent == nil {
		return nil
	}
	return &agentdomain.Agent{
		ID:                agent.ID,
		Name:              agent.Name,
		APIKey:            agent.APIKey,
		Status:            agent.Status,
		Hostname:          agent.Hostname,
		IPAddress:         agent.IPAddress,
		Version:           agent.Version,
		MaxTasks:          agent.MaxTasks,
		CPUThreshold:      agent.CPUThreshold,
		MemThreshold:      agent.MemThreshold,
		DiskThreshold:     agent.DiskThreshold,
		HealthState:       agent.HealthState,
		HealthReason:      agent.HealthReason,
		HealthMessage:     agent.HealthMessage,
		HealthSince:       timeutil.ToUTCPtr(agent.HealthSince),
		RegistrationToken: agent.RegistrationToken,
		ConnectedAt:       timeutil.ToUTCPtr(agent.ConnectedAt),
		LastHeartbeat:     timeutil.ToUTCPtr(agent.LastHeartbeat),
		CreatedAt:         timeutil.ToUTC(agent.CreatedAt),
		UpdatedAt:         timeutil.ToUTC(agent.UpdatedAt),
	}
}

func domainAgentToModel(agent *agentdomain.Agent) *model.Agent {
	if agent == nil {
		return nil
	}
	return &model.Agent{
		ID:                agent.ID,
		Name:              agent.Name,
		APIKey:            agent.APIKey,
		Status:            agent.Status,
		Hostname:          agent.Hostname,
		IPAddress:         agent.IPAddress,
		Version:           agent.Version,
		MaxTasks:          agent.MaxTasks,
		CPUThreshold:      agent.CPUThreshold,
		MemThreshold:      agent.MemThreshold,
		DiskThreshold:     agent.DiskThreshold,
		HealthState:       agent.HealthState,
		HealthReason:      agent.HealthReason,
		HealthMessage:     agent.HealthMessage,
		HealthSince:       timeutil.ToUTCPtr(agent.HealthSince),
		RegistrationToken: agent.RegistrationToken,
		ConnectedAt:       timeutil.ToUTCPtr(agent.ConnectedAt),
		LastHeartbeat:     timeutil.ToUTCPtr(agent.LastHeartbeat),
		CreatedAt:         timeutil.ToUTC(agent.CreatedAt),
		UpdatedAt:         timeutil.ToUTC(agent.UpdatedAt),
	}
}

func modelTokenToDomain(token *model.RegistrationToken) *agentdomain.RegistrationToken {
	if token == nil {
		return nil
	}
	return &agentdomain.RegistrationToken{
		ID:        token.ID,
		Token:     token.Token,
		ExpiresAt: timeutil.ToUTC(token.ExpiresAt),
		CreatedAt: timeutil.ToUTC(token.CreatedAt),
	}
}

func domainTokenToModel(token *agentdomain.RegistrationToken) *model.RegistrationToken {
	if token == nil {
		return nil
	}
	return &model.RegistrationToken{
		ID:        token.ID,
		Token:     token.Token,
		ExpiresAt: timeutil.ToUTC(token.ExpiresAt),
		CreatedAt: timeutil.ToUTC(token.CreatedAt),
	}
}
