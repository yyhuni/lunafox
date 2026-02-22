package application

import (
	"context"
	"errors"
)

func dispatchLogStartedMessage(
	service *AgentRuntimeService,
	_ context.Context,
	_ int,
	message RuntimeMessageInput,
) error {
	if message.LogStarted == nil {
		return errors.New("log_started payload is required")
	}
	return service.handleLogStarted(*message.LogStarted)
}

func dispatchLogChunkMessage(
	service *AgentRuntimeService,
	_ context.Context,
	_ int,
	message RuntimeMessageInput,
) error {
	if message.LogChunk == nil {
		return errors.New("log_chunk payload is required")
	}
	return service.handleLogChunk(*message.LogChunk)
}

func dispatchLogEndMessage(
	service *AgentRuntimeService,
	_ context.Context,
	_ int,
	message RuntimeMessageInput,
) error {
	if message.LogEnd == nil {
		return errors.New("log_end payload is required")
	}
	return service.handleLogEnd(*message.LogEnd)
}

func dispatchLogErrorMessage(
	service *AgentRuntimeService,
	_ context.Context,
	_ int,
	message RuntimeMessageInput,
) error {
	if message.LogError == nil {
		return errors.New("log_error payload is required")
	}
	return service.handleLogError(*message.LogError)
}
