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

func AttachAgentKey(ctx context.Context, value string) context.Context {
	return appendOutgoingMetadata(ctx, AgentKeyHeader, value)
}

func AttachWorkerToken(ctx context.Context, value string) context.Context {
	return appendOutgoingMetadata(ctx, WorkerTokenHeader, value)
}

func AttachTaskToken(ctx context.Context, value string) context.Context {
	return appendOutgoingMetadata(ctx, TaskTokenHeader, value)
}

func ReadIncomingToken(ctx context.Context, key string) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	values := md.Get(strings.ToLower(strings.TrimSpace(key)))
	if len(values) == 0 {
		return "", false
	}
	value := strings.TrimSpace(values[0])
	if value == "" {
		return "", false
	}
	return value, true
}

func appendOutgoingMetadata(ctx context.Context, key, value string) context.Context {
	key = strings.ToLower(strings.TrimSpace(key))
	value = strings.TrimSpace(value)
	if key == "" || value == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, key, value)
}
