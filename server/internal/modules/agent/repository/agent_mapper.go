package repository

import (
	"encoding/json"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
	"gorm.io/datatypes"
)

func modelAgentToDomain(agent *model.Agent) *agentdomain.Agent {
	if agent == nil {
		return nil
	}
	domain := &agentdomain.Agent{
		ID:                  agent.ID,
		InstanceID:          agent.InstanceID,
		DisplayName:         agent.DisplayName,
		AuthenticationToken: agent.AuthenticationToken,
		Status:              agent.Status,
		MaxTasks:            agent.MaxTasks,
		CPUThreshold:        agent.CPUThreshold,
		MemThreshold:        agent.MemThreshold,
		DiskThreshold:       agent.DiskThreshold,
		RegistrationTokenID: agent.RegistrationTokenID,
		CreatedAt:           timeutil.ToUTC(agent.CreatedAt),
		UpdatedAt:           timeutil.ToUTC(agent.UpdatedAt),
	}
	if agent.RuntimeStatus != nil {
		domain.ObservedHostname = agent.RuntimeStatus.ObservedHostname
		domain.ConnectionIP = agent.RuntimeStatus.ConnectionIP
		domain.ObservedSourceIP = agent.RuntimeStatus.ObservedSourceIP
		domain.ObservedIPGeneration = agent.RuntimeStatus.ObservedIPGeneration
		domain.AgentVersion = agent.RuntimeStatus.AgentVersion
		domain.OperatingSystem = agent.RuntimeStatus.OperatingSystem
		domain.Architecture = agent.RuntimeStatus.Architecture
		domain.ContainerRuntimeReady = agent.RuntimeStatus.ContainerRuntimeReady
		_ = json.Unmarshal(agent.RuntimeStatus.SupportedEngineAPIMajors, &domain.SupportedEngineAPIMajors)
		domain.RunningTasks = agent.RuntimeStatus.RunningTasks
		domain.TaskSlotsUsed = agent.RuntimeStatus.TaskSlotsUsed
		domain.SessionID = agent.RuntimeStatus.SessionID
		domain.SessionEpoch = agent.RuntimeStatus.SessionEpoch
		domain.HealthState = agent.RuntimeStatus.HealthState
		domain.HealthReason = agent.RuntimeStatus.HealthReason
		domain.HealthMessage = agent.RuntimeStatus.HealthMessage
		domain.HealthSince = timeutil.ToUTCPtr(agent.RuntimeStatus.HealthSince)
		domain.ConnectedAt = timeutil.ToUTCPtr(agent.RuntimeStatus.ConnectedAt)
		domain.LastHeartbeat = timeutil.ToUTCPtr(agent.RuntimeStatus.LastHeartbeat)
	}
	if agent.Location != nil {
		domain.Location = modelAgentLocationToDomain(agent.Location)
	}
	return domain
}

func domainAgentToModel(agent *agentdomain.Agent) *model.Agent {
	if agent == nil {
		return nil
	}
	return &model.Agent{
		ID:                  agent.ID,
		InstanceID:          agent.InstanceID,
		DisplayName:         firstNonEmpty(agent.DisplayName),
		AuthenticationToken: agent.AuthenticationToken,
		Status:              agent.Status,
		MaxTasks:            agent.MaxTasks,
		CPUThreshold:        agent.CPUThreshold,
		MemThreshold:        agent.MemThreshold,
		DiskThreshold:       agent.DiskThreshold,
		RegistrationTokenID: agent.RegistrationTokenID,
		CreatedAt:           timeutil.ToUTC(agent.CreatedAt),
		UpdatedAt:           timeutil.ToUTC(agent.UpdatedAt),
	}
}

func domainAgentRuntimeStatusToModel(agent *agentdomain.Agent) *model.AgentRuntimeStatus {
	if agent == nil || agent.ID <= 0 {
		return nil
	}
	if agent.ObservedHostname == "" &&
		agent.ConnectionIP == "" &&
		agent.AgentVersion == "" &&
		agent.ConnectedAt == nil &&
		agent.LastHeartbeat == nil &&
		agent.HealthState == "" &&
		agent.HealthReason == "" &&
		agent.HealthMessage == "" &&
		agent.HealthSince == nil {
		return nil
	}
	return &model.AgentRuntimeStatus{
		AgentID:                  agent.ID,
		ObservedHostname:         agent.ObservedHostname,
		ConnectionIP:             agent.ConnectionIP,
		AgentVersion:             agent.AgentVersion,
		ConnectedAt:              timeutil.ToUTCPtr(agent.ConnectedAt),
		LastHeartbeat:            timeutil.ToUTCPtr(agent.LastHeartbeat),
		HealthState:              agent.HealthState,
		HealthReason:             agent.HealthReason,
		HealthMessage:            agent.HealthMessage,
		HealthSince:              timeutil.ToUTCPtr(agent.HealthSince),
		SupportedEngineAPIMajors: datatypes.JSON("[]"),
	}
}

func modelTokenToDomain(token *model.RegistrationToken) *agentdomain.RegistrationToken {
	if token == nil {
		return nil
	}
	return &agentdomain.RegistrationToken{
		ID:               token.ID,
		Token:            token.Token,
		ExpiresAt:        timeutil.ToUTC(token.ExpiresAt),
		EverAttributedAt: timeutil.ToUTCPtr(token.EverAttributedAt),
		CreatedAt:        timeutil.ToUTC(token.CreatedAt),
	}
}

func domainTokenToModel(token *agentdomain.RegistrationToken) *model.RegistrationToken {
	if token == nil {
		return nil
	}
	return &model.RegistrationToken{
		ID:               token.ID,
		Token:            token.Token,
		ExpiresAt:        timeutil.ToUTC(token.ExpiresAt),
		EverAttributedAt: timeutil.ToUTCPtr(token.EverAttributedAt),
		CreatedAt:        timeutil.ToUTC(token.CreatedAt),
	}
}

func modelAgentLocationToDomain(location *model.AgentLocation) *agentdomain.AgentLocationSnapshot {
	if location == nil {
		return nil
	}
	return &agentdomain.AgentLocationSnapshot{
		AgentID:          location.AgentID,
		Latitude:         location.Latitude,
		Longitude:        location.Longitude,
		AccuracyRadiusKM: copyFloat64Ptr(location.AccuracyRadiusKM),
		SourceObservedIP: location.SourceObservedIP,
		ProviderKey:      location.ProviderKey,
		ResolvedAt:       timeutil.ToUTC(location.ResolvedAt),
		ForcedExpired:    location.ForcedExpired,
		UpdatedAt:        timeutil.ToUTC(location.UpdatedAt),
	}
}

func domainAgentLocationToModel(location *agentdomain.AgentLocationSnapshot) *model.AgentLocation {
	if location == nil {
		return nil
	}
	return &model.AgentLocation{
		AgentID:          location.AgentID,
		Latitude:         location.Latitude,
		Longitude:        location.Longitude,
		AccuracyRadiusKM: copyFloat64Ptr(location.AccuracyRadiusKM),
		SourceObservedIP: location.SourceObservedIP,
		ProviderKey:      location.ProviderKey,
		ResolvedAt:       timeutil.ToUTC(location.ResolvedAt),
		ForcedExpired:    location.ForcedExpired,
		UpdatedAt:        timeutil.ToUTC(location.UpdatedAt),
	}
}

func copyFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
