package application

import agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"

// AgentMessagePublisher emits typed messages to agent control-plane connections.
type AgentMessagePublisher interface {
	SendConfigUpdate(agentID int, payload agentdomain.ConfigUpdatePayload)
	SendUpdateRequired(agentID int, payload agentdomain.UpdateRequiredPayload) bool
	SendTaskCancel(agentID, scanID, taskID int)
	TrySendTaskCancel(agentID, scanID, taskID int) bool
}
