package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Progress events are an operator-facing observation channel. They are kept
// deliberately smaller than diagnostics so a journal can never become a raw
// Compose-output transport.
const (
	MaxProgressEvents             = 32
	MaxProgressEventMessageBytes  = 512
	MaxProgressEventMessageKey    = 64
	MaxProgressEventMetadata      = 8
	MaxProgressEventMetadataKey   = 64
	MaxProgressEventMetadataValue = 128
)

// ProgressEvent is the bounded, non-secret sub-step evidence shown by the
// upgrade status page. StageTimes remain authoritative for lifecycle state.
type ProgressEvent struct {
	Timestamp  time.Time         `json:"timestamp"`
	Stage      Status            `json:"stage"`
	MessageKey string            `json:"messageKey"`
	Message    string            `json:"message"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// Validate rejects values that could turn the progress channel into a host
// output, path, or credential exfiltration path.
func (event ProgressEvent) Validate() error {
	if event.Timestamp.IsZero() {
		return fmt.Errorf("progress event timestamp is required")
	}
	if !event.Stage.Valid() {
		return fmt.Errorf("unsupported progress event stage %q", event.Stage)
	}
	if !validProgressToken(event.MessageKey, MaxProgressEventMessageKey) {
		return fmt.Errorf("progress event messageKey is not canonical")
	}
	if err := validateSafeProgressText(event.Message, MaxProgressEventMessageBytes, "message"); err != nil {
		return err
	}
	if len(event.Metadata) > MaxProgressEventMetadata {
		return fmt.Errorf("progress event metadata exceeds %d entries", MaxProgressEventMetadata)
	}
	for key, value := range event.Metadata {
		if !validProgressToken(key, MaxProgressEventMetadataKey) {
			return fmt.Errorf("progress event metadata key %q is not canonical", key)
		}
		if isForbiddenProgressToken(key) {
			return fmt.Errorf("progress event metadata key %q is forbidden", key)
		}
		if err := validateSafeProgressText(value, MaxProgressEventMetadataValue, "metadata value"); err != nil {
			return err
		}
	}
	return nil
}

func validateSafeProgressText(value string, maximumBytes int, field string) error {
	if value == "" || len([]byte(value)) > maximumBytes {
		return fmt.Errorf("progress event %s is empty or exceeds %d bytes", field, maximumBytes)
	}
	if !utf8.ValidString(value) || strings.ContainsAny(value, "\x00\r\n\x1b") {
		return fmt.Errorf("progress event %s contains invalid control characters", field)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("progress event %s contains invalid control characters", field)
		}
	}
	if isForbiddenProgressText(value) {
		return fmt.Errorf("progress event %s contains a forbidden marker", field)
	}
	// A slash/backslash is not needed by any fixed operator sentence and is a
	// useful second fence against leaking deployment or host paths.
	if strings.ContainsAny(value, "/\\`$") {
		return fmt.Errorf("progress event %s contains a path or shell marker", field)
	}
	return nil
}

func isForbiddenProgressText(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{
		"authorization", "bearer ", "jwt", "password", "passwd", "secret", "token",
		"private key", "-----begin", "docker compose", "command:", "stderr:",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func isForbiddenProgressToken(value string) bool {
	return isForbiddenProgressText(value)
}

func validProgressToken(value string, maximumLength int) bool {
	if value == "" || len(value) > maximumLength {
		return false
	}
	for index, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-' {
			if index == 0 && r >= '0' && r <= '9' {
				return false
			}
			continue
		}
		return false
	}
	return true
}

// ValidateProgressEvents validates the persisted collection without changing
// its order. Callers that merge events should use MergeProgressEvents, which
// canonicalizes order and applies the retention bound.
func ValidateProgressEvents(events []ProgressEvent) error {
	if len(events) > MaxProgressEvents {
		return fmt.Errorf("progress events exceed %d entries", MaxProgressEvents)
	}
	var previous time.Time
	for index, event := range events {
		if err := event.Validate(); err != nil {
			return fmt.Errorf("progress event %d: %w", index, err)
		}
		if !previous.IsZero() && event.Timestamp.Before(previous) {
			return fmt.Errorf("progress events are not ordered")
		}
		previous = event.Timestamp
	}
	return nil
}

// MergeProgressEvents appends only new event identities, orders the result by
// host timestamp, and retains the newest bounded window. It is idempotent for
// replayed host journals.
func MergeProgressEvents(existing, incoming []ProgressEvent) ([]ProgressEvent, bool, error) {
	if err := ValidateProgressEvents(existing); err != nil {
		return nil, false, err
	}
	for index, event := range incoming {
		if err := event.Validate(); err != nil {
			return nil, false, fmt.Errorf("incoming progress event %d: %w", index, err)
		}
	}
	merged := cloneProgressEvents(existing)
	seen := make(map[string]struct{}, len(merged)+len(incoming))
	for _, event := range merged {
		seen[progressEventIdentity(event)] = struct{}{}
	}
	changed := false
	for _, event := range incoming {
		identity := progressEventIdentity(event)
		if _, duplicate := seen[identity]; duplicate {
			continue
		}
		seen[identity] = struct{}{}
		merged = append(merged, cloneProgressEvent(event))
		changed = true
	}
	if !changed {
		return cloneProgressEvents(existing), false, nil
	}
	sort.SliceStable(merged, func(left, right int) bool {
		return merged[left].Timestamp.Before(merged[right].Timestamp)
	})
	if len(merged) > MaxProgressEvents {
		merged = merged[len(merged)-MaxProgressEvents:]
	}
	if progressEventsEqual(existing, merged) {
		return cloneProgressEvents(existing), false, nil
	}
	return merged, true, nil
}

func progressEventsEqual(left, right []ProgressEvent) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if progressEventIdentity(left[index]) != progressEventIdentity(right[index]) {
			return false
		}
	}
	return true
}

func cloneProgressEvent(event ProgressEvent) ProgressEvent {
	copy := event
	if event.Metadata != nil {
		copy.Metadata = make(map[string]string, len(event.Metadata))
		for key, value := range event.Metadata {
			copy.Metadata[key] = value
		}
	}
	return copy
}

func cloneProgressEvents(events []ProgressEvent) []ProgressEvent {
	if len(events) == 0 {
		return []ProgressEvent{}
	}
	result := make([]ProgressEvent, len(events))
	for index, event := range events {
		result[index] = cloneProgressEvent(event)
	}
	return result
}

func progressEventIdentity(event ProgressEvent) string {
	keys := make([]string, 0, len(event.Metadata))
	for key := range event.Metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	builder.WriteString(event.Timestamp.UTC().Format(time.RFC3339Nano))
	builder.WriteByte('\x00')
	builder.WriteString(string(event.Stage))
	builder.WriteByte('\x00')
	builder.WriteString(event.MessageKey)
	builder.WriteByte('\x00')
	builder.WriteString(event.Message)
	for _, key := range keys {
		builder.WriteByte('\x00')
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(event.Metadata[key])
	}
	return builder.String()
}

// CloneProgressEvents returns a defensive copy for cross-layer projections.
func CloneProgressEvents(events []ProgressEvent) []ProgressEvent {
	return cloneProgressEvents(events)
}
