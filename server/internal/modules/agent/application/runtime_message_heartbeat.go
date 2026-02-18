package application

import (
	"context"
	"errors"
)

func dispatchHeartbeatMessage(
	service *AgentRuntimeService,
	ctx context.Context,
	agentID int,
	message RuntimeMessageInput,
) error {
	if message.Heartbeat == nil {
		return errors.New("heartbeat payload is required")
	}
	return service.handleHeartbeat(ctx, agentID, *message.Heartbeat)
}
