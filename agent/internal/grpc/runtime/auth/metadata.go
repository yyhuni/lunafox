package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	AgentKeyHeader    = "x-agent-key"
	WorkerTokenHeader = "x-worker-token"
	TaskTokenHeader   = "x-task-token"
)

func WithAgentKey(ctx context.Context, value string) context.Context {
	return withOutgoingMetadata(ctx, AgentKeyHeader, value)
}

func WithWorkerToken(ctx context.Context, value string) context.Context {
	return withOutgoingMetadata(ctx, WorkerTokenHeader, value)
}

func WithTaskToken(ctx context.Context, value string) context.Context {
	return withOutgoingMetadata(ctx, TaskTokenHeader, value)
}

func withOutgoingMetadata(ctx context.Context, key, value string) context.Context {
	key = strings.ToLower(strings.TrimSpace(key))
	value = strings.TrimSpace(value)
	if key == "" || value == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, key, value)
}
