package application

import "github.com/yyhuni/lunafox/server/internal/agentproto"

// AgentMessagePublisher emits typed messages to runtime connections.
type AgentMessagePublisher interface {
	SendConfigUpdate(agentID int, payload agentproto.ConfigUpdatePayload)
	SendUpdateRequired(agentID int, payload agentproto.UpdateRequiredPayload) bool
	SendTaskCancel(agentID, taskID int)
	SendLogOpen(agentID int, payload agentproto.LogOpenPayload) bool
	SendLogCancel(agentID int, payload agentproto.LogCancelPayload) bool
}
