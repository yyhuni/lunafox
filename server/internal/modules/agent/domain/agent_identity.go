package domain

import (
	"fmt"
	"strings"
)

func NormalizeObservedHostname(hostname string) string {
	normalized := strings.TrimSpace(hostname)
	switch strings.ToLower(normalized) {
	case "", "unknown", "localhost":
		return ""
	default:
		return normalized
	}
}

func ShortInstanceID(instanceID string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(instanceID), "-", "")
	if normalized == "" {
		return "unknown"
	}
	if len(normalized) > 6 {
		return normalized[:6]
	}
	return normalized
}

func BuildDefaultAgentDisplayName(observedHostname, instanceID string) string {
	shortID := ShortInstanceID(instanceID)
	normalizedHost := NormalizeObservedHostname(observedHostname)
	if normalizedHost == "" {
		return fmt.Sprintf("agent-%s", shortID)
	}
	return fmt.Sprintf("%s-%s", normalizedHost, shortID)
}
