package application

import agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"

// AgentMessagePublisher emits typed messages to runtime connections.
type AgentMessagePublisher interface {
	SendConfigUpdate(agentID int, payload agentdomain.ConfigUpdatePayload)
	SendUpdateRequired(agentID int, payload agentdomain.UpdateRequiredPayload) bool
	SendTaskCancel(agentID, taskID int)
}
