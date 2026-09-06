package auth

import (
	"context"
	"strconv"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	AgentAuthenticationTokenMetadataKey = "lunafox-agent-authentication-token"
	AgentSessionIDMetadataKey           = "lunafox-agent-session-id"
	AgentSessionEpochMetadataKey        = "lunafox-agent-session-epoch"
)

func WithAgentAuthenticationToken(ctx context.Context, value string) context.Context {
	return withOutgoingMetadata(ctx, AgentAuthenticationTokenMetadataKey, value)
}

func ReadAgentAuthenticationToken(ctx context.Context) (string, bool) {
	return readIncomingMetadata(ctx, AgentAuthenticationTokenMetadataKey)
}

func WithAgentSession(ctx context.Context, sessionID string, sessionEpoch int64) context.Context {
	if sessionEpoch <= 0 {
		return ctx
	}
	ctx = withOutgoingMetadata(ctx, AgentSessionIDMetadataKey, sessionID)
	return withOutgoingMetadata(ctx, AgentSessionEpochMetadataKey, strconv.FormatInt(sessionEpoch, 10))
}

// ReadAgentSession requires one canonical value for each half of the fencing
// tuple. Duplicate metadata is ambiguous and therefore fails closed.
func ReadAgentSession(ctx context.Context) (sessionID string, sessionEpoch int64, ok bool) {
	sessionID, sessionIDOK := readSingleIncomingMetadata(ctx, AgentSessionIDMetadataKey)
	epochValue, epochOK := readSingleIncomingMetadata(ctx, AgentSessionEpochMetadataKey)
	if !sessionIDOK || !epochOK {
		return "", 0, false
	}
	parsed, err := strconv.ParseInt(epochValue, 10, 64)
	if err != nil || parsed <= 0 || strconv.FormatInt(parsed, 10) != epochValue {
		return "", 0, false
	}
	return sessionID, parsed, true
}

func readIncomingMetadata(ctx context.Context, key string) (string, bool) {
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

func readSingleIncomingMetadata(ctx context.Context, key string) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	values := md.Get(strings.ToLower(strings.TrimSpace(key)))
	if len(values) != 1 {
		return "", false
	}
	raw := values[0]
	value := strings.TrimSpace(raw)
	if value == "" || value != raw {
		return "", false
	}
	return value, true
}

func withOutgoingMetadata(ctx context.Context, key, value string) context.Context {
	key = strings.ToLower(strings.TrimSpace(key))
	value = strings.TrimSpace(value)
	if key == "" || value == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, key, value)
}
