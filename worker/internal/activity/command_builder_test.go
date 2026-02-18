package activity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandBuilder_Build_BasicTemplate(t *testing.T) {
	builder := NewCommandBuilder()

	tmpl := CommandTemplate{
		BaseCommand: "subfinder -d {{.Domain}} -o {{quote .OutputFile}}",
		CLIParams:   []Parameter{},
	}

	params := map[string]any{
		"Domain":     "example.com",
		"OutputFile": "/tmp/output.txt",
	}

	cmd, err := builder.Build(tmpl, params, nil)
	require.NoError(t, err)
	assert.Equal(t, `subfinder -d example.com -o "/tmp/output.txt"`, cmd)
}

func TestCommandBuilder_Build_WithoutDefaults(t *testing.T) {
	builder := NewCommandBuilder()

	tmpl := CommandTemplate{
		BaseCommand: "subfinder -d {{.Domain}}",
		CLIParams: []Parameter{
			{
				SemanticID: "timeout-cli",
				Var:        "Timeout",
				Arg:        "-timeout {{.Timeout}}",
				ConfigSchema: ConfigSchema{
					Key:      "timeout-cli",
					Type:     "integer",
					Required: false,
				},
				ConfigExample: ConfigExample{
					ShowAs: "value",
				},
				Documentation: Documentation{
					Description: "Scan timeout in seconds",
				},
			},
			{
				SemanticID: "threads-cli",
				Var:        "Threads",
				Arg:        "-t {{.Threads}}",
				ConfigSchema: ConfigSchema{
					Key:      "threads-cli",
					Type:     "integer",
					Required: false,
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

	params := map[string]any{
		"Domain": "example.com",
	}

	cmd, err := builder.Build(tmpl, params, nil)
	require.NoError(t, err)
	assert.Contains(t, cmd, "subfinder -d example.com")
	assert.NotContains(t, cmd, "-timeout")
	assert.NotContains(t, cmd, "-t ")
}

func TestCommandBuilder_Build_UserConfigProvided(t *testing.T) {
	builder := NewCommandBuilder()

	tmpl := CommandTemplate{
		BaseCommand: "subfinder -d {{.Domain}}",
		CLIParams: []Parameter{
			{
				SemanticID: "timeout-cli",
				Var:        "Timeout",
				Arg:        "-timeout {{.Timeout}}",
				ConfigSchema: ConfigSchema{
					Key:      "timeout-cli",
					Type:     "integer",
					Required: true,
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

	params := map[string]any{
		"Domain": "example.com",
	}

	config := map[string]any{
		"timeout-cli": 7200,
	}

	cmd, err := builder.Build(tmpl, params, config)
	require.NoError(t, err)
	assert.Contains(t, cmd, "-timeout 7200")
}

func TestCommandBuilder_Build_OptionalParameterWithoutDefault(t *testing.T) {
	builder := NewCommandBuilder()

	tmpl := CommandTemplate{
		BaseCommand: "subfinder -d {{.Domain}}",
		CLIParams: []Parameter{
			{
				SemanticID: "provider-config-cli",
				Var:        "ProviderConfig",
				Arg:        "-pc {{quote .ProviderConfig}}",
				ConfigSchema: ConfigSchema{
					Key:      "provider-config-cli",
					Type:     "string",
					Required: false,
				},
				ConfigExample: ConfigExample{
					ShowAs: "comment",
				},
				Documentation: Documentation{
					Description: "Provider configuration file path",
				},
			},
		},
	}

	params := map[string]any{
		"Domain": "example.com",
	}
	// Don't provide ProviderConfig, should not add this parameter
	cmd, err := builder.Build(tmpl, params, nil)
	require.NoError(t, err)
	assert.Equal(t, "subfinder -d example.com", cmd)
	assert.NotContains(t, cmd, "-pc")
}

func TestCommandBuilder_Build_OptionalParameterWithUserValue(t *testing.T) {
	builder := NewCommandBuilder()

	tmpl := CommandTemplate{
		BaseCommand: "subfinder -d {{.Domain}}",
		CLIParams: []Parameter{
			{
				SemanticID: "provider-config-cli",
				Var:        "ProviderConfig",
				Arg:        "-pc {{quote .ProviderConfig}}",
				ConfigSchema: ConfigSchema{
					Key:      "provider-config-cli",
					Type:     "string",
					Required: false,
				},
				ConfigExample: ConfigExample{
					ShowAs: "comment",
				},
				Documentation: Documentation{
					Description: "Provider configuration file path",
				},
			},
		},
	}

	params := map[string]any{
		"Domain": "example.com",
	}

	config := map[string]any{
		"provider-config-cli": "/etc/config.yaml",
	}

	cmd, err := builder.Build(tmpl, params, config)
	require.NoError(t, err)
	assert.Contains(t, cmd, `-pc "/etc/config.yaml"`)
}

func TestCommandBuilder_Build_MissingRequiredParameter(t *testing.T) {
	builder := NewCommandBuilder()

	tmpl := CommandTemplate{
		BaseCommand: "subfinder -d {{.Domain}}",
		CLIParams:   []Parameter{},
	}

	params := map[string]any{
		// Domain is missing
	}

	_, err := builder.Build(tmpl, params, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Domain")
}

func TestCommandBuilder_Build_MissingRequiredConfigParam(t *testing.T) {
	builder := NewCommandBuilder()

	tmpl := CommandTemplate{
		BaseCommand: "subfinder -d {{.Domain}}",
		CLIParams: []Parameter{
			{
				SemanticID: "timeout-cli",
				Var:        "Timeout",
				Arg:        "-timeout {{.Timeout}}",
				ConfigSchema: ConfigSchema{
					Key:      "timeout-cli",
					Type:     "integer",
					Required: true,
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

	params := map[string]any{
		"Domain": "example.com",
	}

	_, err := builder.Build(tmpl, params, map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required parameter")
}

func TestCommandBuilder_Build_InvalidTemplate(t *testing.T) {
	builder := NewCommandBuilder()

	tmpl := CommandTemplate{
		BaseCommand: "tool {{.Domain",
	}

	_, err := builder.Build(tmpl, map[string]any{"Domain": "example.com"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestCommandBuilder_Build_MissingTemplateVar(t *testing.T) {
	builder := NewCommandBuilder()

	tmpl := CommandTemplate{
		BaseCommand: "tool {{.Missing}}",
	}

	_, err := builder.Build(tmpl, map[string]any{}, nil)
	require.Error(t, err)
}

func TestCommandBuilder_Build_TypeValidation(t *testing.T) {
	builder := NewCommandBuilder()

	tmpl := CommandTemplate{
		BaseCommand: "subfinder -d {{.Domain}}",
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

	params := map[string]any{
		"Domain": "example.com",
	}

	config := map[string]any{
		"timeout-cli": "not-an-int", // Wrong type
	}

	_, err := builder.Build(tmpl, params, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Timeout")
}

func TestCommandBuilder_Build_QuoteFunction(t *testing.T) {
	builder := NewCommandBuilder()

	tmpl := CommandTemplate{
		BaseCommand: "cat {{quote .InputFile}} | tool",
		CLIParams:   []Parameter{},
	}

	params := map[string]any{
		"InputFile": "/path/with spaces/file.txt",
	}

	cmd, err := builder.Build(tmpl, params, nil)
	require.NoError(t, err)
	assert.Contains(t, cmd, `"/path/with spaces/file.txt"`)
}

func TestCommandBuilder_Build_BoolParameter(t *testing.T) {
	builder := NewCommandBuilder()

	tmpl := CommandTemplate{
		BaseCommand: "tool {{if .Verbose}}-v{{end}}",
		CLIParams: []Parameter{
			{
				SemanticID: "verbose-cli",
				Var:        "Verbose",
				Arg:        "",
				ConfigSchema: ConfigSchema{
					Key:      "verbose-cli",
					Type:     "boolean",
					Required: true,
				},
				ConfigExample: ConfigExample{
					ShowAs: "value",
				},
				Documentation: Documentation{
					Description: "Enable verbose output",
				},
			},
		},
	}

	params := map[string]any{}
	_, err := builder.Build(tmpl, params, nil)
	require.Error(t, err)

	// Verbose = true
	config := map[string]any{"verbose-cli": true}
	cmd, err := builder.Build(tmpl, params, config)
	require.NoError(t, err)
	assert.Equal(t, "tool -v", cmd)

	// Verbose = false (explicit)
	config = map[string]any{"verbose-cli": false}
	cmd, err = builder.Build(tmpl, params, config)
	require.NoError(t, err)
	assert.Equal(t, "tool", cmd)
}

func TestCommandBuilder_Build_ComplexTemplate(t *testing.T) {
	builder := NewCommandBuilder()

	tmpl := CommandTemplate{
		BaseCommand: "puredns bruteforce {{quote .Wordlist}} {{.Domain}} -r {{quote .Resolvers}} --write {{quote .OutputFile}}",
		CLIParams: []Parameter{
			{
				SemanticID: "threads-cli",
				Var:        "Threads",
				Arg:        "-t {{.Threads}}",
				ConfigSchema: ConfigSchema{
					Key:      "threads-cli",
					Type:     "integer",
					Required: true,
				},
				ConfigExample: ConfigExample{
					ShowAs: "value",
				},
				Documentation: Documentation{
					Description: "Number of concurrent threads",
				},
			},
			{
				SemanticID: "rate-limit-cli",
				Var:        "RateLimit",
				Arg:        "--rate-limit {{.RateLimit}}",
				ConfigSchema: ConfigSchema{
					Key:      "rate-limit-cli",
					Type:     "integer",
					Required: true,
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

	params := map[string]any{
		"Domain":     "example.com",
		"Wordlist":   "/wordlists/subdomains.txt",
		"Resolvers":  "/etc/resolvers.txt",
		"OutputFile": "/tmp/output.txt",
	}

	config := map[string]any{
		"threads-cli":    200,
		"rate-limit-cli": 150,
	}

	cmd, err := builder.Build(tmpl, params, config)
	require.NoError(t, err)
	assert.Contains(t, cmd, "puredns bruteforce")
	assert.Contains(t, cmd, `"/wordlists/subdomains.txt"`)
	assert.Contains(t, cmd, "example.com")
	assert.Contains(t, cmd, "-t 200")
	assert.Contains(t, cmd, "--rate-limit 150")
}
