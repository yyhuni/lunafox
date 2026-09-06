// Package idempotency contains the request-replay primitives shared by MCP
// mutations. It deliberately has no tool or transport dependency so the
// application services remain the owners of validation and state changes.
package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

const (
	// MaxRequestIDLength bounds a caller-provided business replay key without
	// conflating it with the transport Request-Id audit correlation header.
	MaxRequestIDLength = 128
	// ReplayRetention is the fixed bounded replay window shared by all MCP
	// side-effect commands. It is deliberately not caller-configurable.
	ReplayRetention = 30 * 24 * time.Hour
	// ReplayPruneLimit bounds opportunistic cleanup work performed inside a
	// business transaction.
	ReplayPruneLimit = 100
)

// NormalizeRequestID accepts an omitted replay key and otherwise returns the
// one stable representation used by every MCP mutation.
func NormalizeRequestID(value string) (string, error) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", nil
	}
	if len(normalized) > MaxRequestIDLength || !utf8.ValidString(normalized) || strings.IndexFunc(normalized, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("request_id must be valid UTF-8 text no longer than %d bytes", MaxRequestIDLength)
	}
	return normalized, nil
}

// Fingerprint canonically serializes an action and its effectful input before
// hashing it. Arrays remain ordered, which preserves batch disposition order.
func Fingerprint(action string, input any) (string, error) {
	if strings.TrimSpace(action) == "" {
		return "", fmt.Errorf("replay action is required")
	}
	raw, err := json.Marshal(struct {
		Action string `json:"action"`
		Input  any    `json:"input"`
	}{Action: strings.TrimSpace(action), Input: input})
	if err != nil {
		return "", fmt.Errorf("marshal replay fingerprint: %w", err)
	}
	canonical, err := jsoncanonicalizer.Transform(raw)
	if err != nil {
		return "", fmt.Errorf("canonicalize replay fingerprint: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}
