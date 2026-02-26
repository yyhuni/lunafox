package auth

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestRuntimeMetadataKeysStable(t *testing.T) {
	if AgentKeyHeader != "x-agent-key" {
		t.Fatalf("unexpected agent key header: %q", AgentKeyHeader)
	}
	if WorkerTokenHeader != "x-worker-token" {
		t.Fatalf("unexpected worker token header: %q", WorkerTokenHeader)
	}
	if TaskTokenHeader != "x-task-token" {
		t.Fatalf("unexpected task token header: %q", TaskTokenHeader)
	}
}

func TestAttachAndExtractMetadata(t *testing.T) {
	ctx := context.Background()
	ctx = AttachAgentKey(ctx, "agent-k")
	ctx = AttachWorkerToken(ctx, "worker-t")
	ctx = AttachTaskToken(ctx, "task-t")

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("expected outgoing metadata")
	}

	if got := firstMD(md, AgentKeyHeader); got != "agent-k" {
		t.Fatalf("unexpected agent key: %q", got)
	}
	if got := firstMD(md, WorkerTokenHeader); got != "worker-t" {
		t.Fatalf("unexpected worker token: %q", got)
	}
	if got := firstMD(md, TaskTokenHeader); got != "task-t" {
		t.Fatalf("unexpected task token: %q", got)
	}
}

func TestReadIncomingToken(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(TaskTokenHeader, "task-123"))

	got, ok := ReadIncomingToken(ctx, TaskTokenHeader)
	if !ok {
		t.Fatalf("expected token present")
	}
	if got != "task-123" {
		t.Fatalf("unexpected token: %q", got)
	}
}

func firstMD(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
