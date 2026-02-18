package application

import (
	"context"
)

type runtimeMessageHandler func(service *AgentRuntimeService, ctx context.Context, agentID int, message RuntimeMessageInput) error

func newRuntimeMessageDispatcher() map[string]runtimeMessageHandler {
	return map[string]runtimeMessageHandler{
		RuntimeMessageTypeHeartbeat: dispatchHeartbeatMessage,
	}
}
