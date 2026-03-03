package handler

import (
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func toAgentOutput(agent *agentdomain.Agent, heartbeat *dto.AgentHeartbeatResponse) dto.AgentResponse {
	return dto.AgentResponse{
		ID:            agent.ID,
		Name:          agent.Name,
		Status:        agent.Status,
		Hostname:      agent.Hostname,
		IPAddress:     agent.IPAddress,
		AgentVersion:  agent.AgentVersion,
		WorkerVersion: agent.WorkerVersion,
		MaxTasks:      agent.MaxTasks,
		CPUThreshold:  agent.CPUThreshold,
		MemThreshold:  agent.MemThreshold,
		DiskThreshold: agent.DiskThreshold,
		ConnectedAt:   timeutil.ToUTCPtr(agent.ConnectedAt),
		LastHeartbeat: timeutil.ToUTCPtr(agent.LastHeartbeat),
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
