package activity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapConfigKeys_MapsConfigKey(t *testing.T) {
	tmpl := CommandTemplate{
		BaseCommand: "tool",
		CLIParams: []Parameter{
			{
				SemanticID: "timeout-cli",
				Var:        "Timeout",
				Arg:        "-timeout {{.Timeout}}",
				ConfigSchema: ConfigSchema{
					Key:  "timeout-cli",
					Type: "integer",
				},
				ConfigExample: ConfigExample{
					ShowAs: "value",
				},
				Documentation: Documentation{
					Description: "Scan timeout in seconds",
				},
			},
		},
	}

	raw := map[string]any{
		"enabled":     true,
		"timeout-cli": 3600,
	}

	normalized, err := MapConfigKeys(tmpl, raw)
	require.NoError(t, err)
	require.Equal(t, 3600, normalized["timeout-cli"])
	require.NotContains(t, normalized, "enabled")
}

func TestMapConfigKeys_RejectsUnknownKey(t *testing.T) {
	tmpl := CommandTemplate{
		BaseCommand: "tool",
		CLIParams: []Parameter{
			{
				SemanticID: "threads-cli",
				Var:        "Threads",
				Arg:        "-t {{.Threads}}",
				ConfigSchema: ConfigSchema{
					Key:  "threads-cli",
					Type: "integer",
				},
				ConfigExample: ConfigExample{
					ShowAs: "value",
				},
				Documentation: Documentation{
					Description: "Number of concurrent threads",
				},
			},
		},
	}

	_, err := MapConfigKeys(tmpl, map[string]any{"unknown": 1})
	require.Error(t, err)
}

func TestMapConfigKeys_TypeValidation(t *testing.T) {
	tmpl := CommandTemplate{
		BaseCommand: "tool",
		CLIParams: []Parameter{
			{
				SemanticID: "rate-limit-cli",
				Var:        "RateLimit",
				Arg:        "--rate-limit {{.RateLimit}}",
				ConfigSchema: ConfigSchema{
					Key:  "rate-limit-cli",
					Type: "integer",
				},
				ConfigExample: ConfigExample{
					ShowAs: "value",
				},
				Documentation: Documentation{
					Description: "Rate limit per second",
				},
			},
		},
	}

	_, err := MapConfigKeys(tmpl, map[string]any{"rate-limit-cli": "bad"})
	require.Error(t, err)
}

func TestMapConfigKeys_InternalParams(t *testing.T) {
	tmpl := CommandTemplate{
		BaseCommand: "tool",
		CLIParams:   []Parameter{},
		InternalParams: map[string]any{
			"subdomain-wordlist-base-path-runtime": "/opt/lunafox/wordlists",
		},
	}

	normalized, err := MapConfigKeys(tmpl, map[string]any{"enabled": true})
	require.NoError(t, err)
	require.Equal(t, "/opt/lunafox/wordlists", normalized["subdomain-wordlist-base-path-runtime"])
}

func TestMapConfigKeys_InternalParams_RejectOverride(t *testing.T) {
	tmpl := CommandTemplate{
		BaseCommand: "tool",
		CLIParams:   []Parameter{},
		InternalParams: map[string]any{
			"subdomain-wordlist-base-path-runtime": "/opt/lunafox/wordlists",
		},
	}

	_, err := MapConfigKeys(tmpl, map[string]any{
		"subdomain-wordlist-base-path-runtime": "/tmp/wordlists",
	})
	require.Error(t, err)
}

func TestMapConfigKeys_NilRawReturnsInternal(t *testing.T) {
	tmpl := CommandTemplate{
		BaseCommand: "tool",
		CLIParams:   []Parameter{},
		InternalParams: map[string]any{
			"subdomain-wordlist-base-path-runtime": "/opt/lunafox/wordlists",
		},
	}

	normalized, err := MapConfigKeys(tmpl, nil)
	require.NoError(t, err)
	require.Equal(t, "/opt/lunafox/wordlists", normalized["subdomain-wordlist-base-path-runtime"])
}

func TestMapConfigKeys_MissingSemanticID(t *testing.T) {
	tmpl := CommandTemplate{
		BaseCommand: "tool",
		CLIParams: []Parameter{
			{
				SemanticID: "",
				Var:        "Timeout",
				Arg:        "-timeout {{.Timeout}}",
				ConfigSchema: ConfigSchema{
					Key:  "timeout-cli",
					Type: "integer",
				},
				ConfigExample: ConfigExample{
					ShowAs: "value",
				},
				Documentation: Documentation{
					Description: "Scan timeout in seconds",
				},
			},
		},
	}

	_, err := MapConfigKeys(tmpl, map[string]any{"timeout-cli": 10})
	require.Error(t, err)
}

func TestMapConfigKeys_SemanticIDConflictsWithInternal(t *testing.T) {
	tmpl := CommandTemplate{
		BaseCommand: "tool",
		CLIParams: []Parameter{
			{
				SemanticID: "timeout-runtime",
				Var:        "Timeout",
				Arg:        "-timeout {{.Timeout}}",
				ConfigSchema: ConfigSchema{
					Key:  "timeout-cli",
					Type: "integer",
				},
				ConfigExample: ConfigExample{
					ShowAs: "value",
				},
				Documentation: Documentation{
					Description: "Scan timeout in seconds",
				},
			},
		},
		InternalParams: map[string]any{
			"timeout-runtime": 30,
		},
	}

	_, err := MapConfigKeys(tmpl, map[string]any{"timeout-cli": 10})
	require.Error(t, err)
}
