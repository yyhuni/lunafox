package auth

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestRuntimeMetadataKeysStable(t *testing.T) {
	if AgentAuthenticationTokenMetadataKey != "lunafox-agent-authentication-token" {
		t.Fatalf("unexpected agent authentication token metadata key: %q", AgentAuthenticationTokenMetadataKey)
	}
	if AgentSessionIDMetadataKey != "lunafox-agent-session-id" || AgentSessionEpochMetadataKey != "lunafox-agent-session-epoch" {
		t.Fatalf("unexpected Agent session metadata keys: %q %q", AgentSessionIDMetadataKey, AgentSessionEpochMetadataKey)
	}
}

func TestWithAndExtractMetadata(t *testing.T) {
	ctx := context.Background()
	ctx = WithAgentAuthenticationToken(ctx, "agent-k")
	ctx = WithAgentSession(ctx, "session-a", 17)

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("expected outgoing metadata")
	}

	if got := firstMD(md, AgentAuthenticationTokenMetadataKey); got != "agent-k" {
		t.Fatalf("unexpected agent authentication token: %q", got)
	}
	if got := firstMD(md, AgentSessionIDMetadataKey); got != "session-a" {
		t.Fatalf("unexpected Agent session ID: %q", got)
	}
	if got := firstMD(md, AgentSessionEpochMetadataKey); got != "17" {
		t.Fatalf("unexpected Agent session epoch: %q", got)
	}
}

func TestReadAgentSessionRequiresCanonicalSingleTuple(t *testing.T) {
	valid := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		AgentSessionIDMetadataKey, "session-a",
		AgentSessionEpochMetadataKey, "17",
	))
	if sessionID, epoch, ok := ReadAgentSession(valid); !ok || sessionID != "session-a" || epoch != 17 {
		t.Fatalf("ReadAgentSession() = %q, %d, %v", sessionID, epoch, ok)
	}
	for _, md := range []metadata.MD{
		metadata.Pairs(AgentSessionIDMetadataKey, "session-a"),
		metadata.Pairs(AgentSessionIDMetadataKey, "session-a", AgentSessionEpochMetadataKey, "017"),
		metadata.Pairs(AgentSessionIDMetadataKey, "session-a", AgentSessionEpochMetadataKey, "0"),
		metadata.Pairs(AgentSessionIDMetadataKey, "session-a", AgentSessionEpochMetadataKey, "1", AgentSessionEpochMetadataKey, "1"),
	} {
		ctx := metadata.NewIncomingContext(context.Background(), md)
		if sessionID, epoch, ok := ReadAgentSession(ctx); ok || sessionID != "" || epoch != 0 {
			t.Fatalf("invalid tuple accepted: %q, %d, %v", sessionID, epoch, ok)
		}
	}
}

func TestReadIncomingMetadata(t *testing.T) {
	agentCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(AgentAuthenticationTokenMetadataKey, "agent-123"))
	authenticationToken, ok := ReadAgentAuthenticationToken(agentCtx)
	if !ok {
		t.Fatalf("expected agent authentication token present")
	}
	if authenticationToken != "agent-123" {
		t.Fatalf("unexpected agent authentication token: %q", authenticationToken)
	}
}

func TestReadIncomingMetadataRejectsMissingOrBlankValues(t *testing.T) {
	t.Run("missing incoming metadata", func(t *testing.T) {
		if got, ok := readIncomingMetadata(context.Background(), AgentAuthenticationTokenMetadataKey); ok || got != "" {
			t.Fatalf("expected no agent authentication token, got=%q ok=%v", got, ok)
		}
	})

	t.Run("blank incoming value", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(AgentAuthenticationTokenMetadataKey, "   "))
		if got, ok := ReadAgentAuthenticationToken(ctx); ok || got != "" {
			t.Fatalf("expected blank agent authentication token to be rejected, got=%q ok=%v", got, ok)
		}
	})
}

func TestMetadataHelpersNormalizeKeyAndValue(t *testing.T) {
	ctx := withOutgoingMetadata(context.Background(), " X-Test-Key ", "  value ")
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("expected outgoing metadata")
	}
	if got := firstMD(md, "x-test-key"); got != "value" {
		t.Fatalf("expected normalized metadata value, got %q", got)
	}

	incoming := metadata.NewIncomingContext(context.Background(), metadata.Pairs("X-Test-Key", "  incoming "))
	if got, ok := readIncomingMetadata(incoming, " x-test-key "); !ok || got != "incoming" {
		t.Fatalf("expected normalized incoming metadata, got=%q ok=%v", got, ok)
	}
}

func TestWithOutgoingMetadataSkipsBlankKeyOrValue(t *testing.T) {
	noKey := withOutgoingMetadata(context.Background(), "   ", "value")
	if _, ok := metadata.FromOutgoingContext(noKey); ok {
		t.Fatalf("expected blank key to skip outgoing metadata")
	}

	noValue := withOutgoingMetadata(context.Background(), "x-test-key", "   ")
	if _, ok := metadata.FromOutgoingContext(noValue); ok {
		t.Fatalf("expected blank value to skip outgoing metadata")
	}
}

func firstMD(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
