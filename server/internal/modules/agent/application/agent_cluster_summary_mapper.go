package application

import agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"

type AgentClusterState string

const (
	AgentClusterStateEmpty          AgentClusterState = "empty"
	AgentClusterStateCritical       AgentClusterState = "critical"
	AgentClusterStateNeedsAttention AgentClusterState = "needsAttention"
	AgentClusterStateHealthy        AgentClusterState = "healthy"
)

type AgentClusterReasonCode string

const (
	AgentClusterReasonNoAgents                 AgentClusterReasonCode = "no_agents"
	AgentClusterReasonNoAvailableSlots         AgentClusterReasonCode = "no_available_slots"
	AgentClusterReasonOfflineAgents            AgentClusterReasonCode = "offline_agents"
	AgentClusterReasonWarningAgents            AgentClusterReasonCode = "warning_agents"
	AgentClusterReasonUnknownAgents            AgentClusterReasonCode = "unknown_agents"
	AgentClusterReasonStaleRuntimeObservations AgentClusterReasonCode = "stale_runtime_observations"
	AgentClusterReasonOvercommittedSlots       AgentClusterReasonCode = "overcommitted_slots"
)

type AgentClusterConclusion struct {
	State       AgentClusterState
	ReasonCodes []AgentClusterReasonCode
}

func mapAgentClusterConclusion(aggregate agentdomain.AgentClusterAggregate) AgentClusterConclusion {
	if aggregate.TotalNodes == 0 {
		return AgentClusterConclusion{
			State:       AgentClusterStateEmpty,
			ReasonCodes: []AgentClusterReasonCode{AgentClusterReasonNoAgents},
		}
	}

	reasons := make([]AgentClusterReasonCode, 0, 6)
	state := AgentClusterStateHealthy
	if aggregate.AvailableSlots == 0 {
		state = AgentClusterStateCritical
		reasons = append(reasons, AgentClusterReasonNoAvailableSlots)
	}
	if aggregate.OfflineCount > 0 {
		reasons = append(reasons, AgentClusterReasonOfflineAgents)
	}
	if aggregate.WarningCount > 0 {
		reasons = append(reasons, AgentClusterReasonWarningAgents)
	}
	if aggregate.UnknownCount > 0 {
		reasons = append(reasons, AgentClusterReasonUnknownAgents)
	}
	if aggregate.StaleAgentCount > 0 {
		reasons = append(reasons, AgentClusterReasonStaleRuntimeObservations)
	}
	if aggregate.OvercommittedSlots > 0 {
		reasons = append(reasons, AgentClusterReasonOvercommittedSlots)
	}
	if state != AgentClusterStateCritical && len(reasons) > 0 {
		state = AgentClusterStateNeedsAttention
	}
	return AgentClusterConclusion{State: state, ReasonCodes: reasons}
}
