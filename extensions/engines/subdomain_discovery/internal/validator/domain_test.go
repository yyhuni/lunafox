package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeSubdomainLineNormalizesToolOutput(t *testing.T) {
	got, ok := NormalizeSubdomainLine("  Api.Example.COM.  ")

	require.True(t, ok)
	require.Equal(t, "api.example.com", got)
}

func TestNormalizeSubdomainLineRejectsNonSubdomainOutput(t *testing.T) {
	tests := []string{
		"",
		"# comment",
		"bad domain",
		"localhost",
		"127.0.0.1",
	}

	for _, tt := range tests {
		got, ok := NormalizeSubdomainLine(tt)
		require.False(t, ok, tt)
		require.Empty(t, got, tt)
	}
}

func TestIsValidSubdomainFormatUsesNormalizedSubdomainLine(t *testing.T) {
	require.True(t, IsValidSubdomainFormat("Api.Example.COM."))
	require.False(t, IsValidSubdomainFormat("bad domain"))
}
