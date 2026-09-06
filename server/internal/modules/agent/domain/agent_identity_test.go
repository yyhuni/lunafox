package domain

import "testing"

func TestNormalizeObservedHostnameTreatsSpecialValuesAsEmpty(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "blank", input: "  ", expected: ""},
		{name: "unknown", input: "unknown", expected: ""},
		{name: "localhost", input: " LOCALHOST ", expected: ""},
		{name: "normal hostname", input: " node-a ", expected: "node-a"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeObservedHostname(tc.input); got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestBuildDefaultAgentDisplayNameUsesHostAndShortID(t *testing.T) {
	got := BuildDefaultAgentDisplayName("node-a", "agt-12345678")

	if got != "node-a-agt123" {
		t.Fatalf("unexpected display name: %q", got)
	}
}

func TestBuildDefaultAgentDisplayNameFallsBackToAgentPrefix(t *testing.T) {
	got := BuildDefaultAgentDisplayName("", "agt-12345678")

	if got != "agent-agt123" {
		t.Fatalf("unexpected fallback display name: %q", got)
	}
}
