package subdomain_discovery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeWorkflowConfigSuccess(t *testing.T) {
	raw := validAuthoritativeConfigMap()

	cfg, err := decodeWorkflowConfig(raw)
	require.NoError(t, err)
	require.Equal(t, ContractAPIVersion, cfg.APIVersion)
	require.Equal(t, ContractSchemaVer, cfg.SchemaVersion)
	require.True(t, cfg.Recon.Enabled)
	require.True(t, cfg.Recon.Tools.Subfinder.Enabled)
}

func TestDecodeWorkflowConfigInvalidType(t *testing.T) {
	raw := validAuthoritativeConfigMap()
	recon := raw[stageRecon].(map[string]any)
	tools := recon["tools"].(map[string]any)
	subfinder := tools[toolSubfinder].(map[string]any)
	subfinder["threads-cli"] = "10"

	_, err := decodeWorkflowConfig(raw)
	require.Error(t, err)
}

func TestValidateAuthoritativeConfig_ServerPassWorkerReject(t *testing.T) {
	raw := validAuthoritativeConfigMap()
	recon := raw[stageRecon].(map[string]any)
	recon["enabled"] = true
	tools := recon["tools"].(map[string]any)
	tools[toolSubfinder] = map[string]any{
		"enabled":         false,
		"timeout-runtime": 3600,
		"threads-cli":     10,
	}

	require.NoError(t, validateExplicitConfig(raw))
	err := validateAuthoritativeConfig(raw)
	require.Error(t, err)
	require.Contains(t, err.Error(), "recon")
}

func TestValidateAuthoritativeConfig_BruteforceEnabledRequiresWordlist(t *testing.T) {
	raw := validAuthoritativeConfigMap()
	bruteforce := raw[stageBruteforce].(map[string]any)
	bruteforce["enabled"] = true
	tools := bruteforce["tools"].(map[string]any)
	tools[toolSubdomainBruteforce] = map[string]any{
		"enabled":                         true,
		"timeout-runtime":                 3600,
		"subdomain-wordlist-name-runtime": "",
		"threads-cli":                     50,
		"rate-limit-cli":                  1000,
		"wildcard-tests-cli":              5,
		"wildcard-batch-cli":              1000000,
	}

	require.NoError(t, validateExplicitConfig(raw))
	err := validateAuthoritativeConfig(raw)
	require.Error(t, err)
	require.Contains(t, err.Error(), "subdomain-wordlist-name-runtime")
}

func validAuthoritativeConfigMap() map[string]any {
	return map[string]any{
		"apiVersion":    ContractAPIVersion,
		"schemaVersion": ContractSchemaVer,
			stageRecon: map[string]any{
				"enabled": true,
				"tools": map[string]any{
					toolSubfinder: map[string]any{
						"enabled":         true,
						"timeout-runtime": 3600,
						"threads-cli":     10,
					},
				},
			},
		stageBruteforce: map[string]any{
			"enabled": false,
			"tools": map[string]any{
				toolSubdomainBruteforce: map[string]any{"enabled": false},
			},
		},
		stagePermutation: map[string]any{
			"enabled": false,
			"tools": map[string]any{
				toolSubdomainPermutationResolve: map[string]any{"enabled": false},
			},
		},
		stageResolve: map[string]any{
			"enabled": false,
			"tools": map[string]any{
				toolSubdomainResolve: map[string]any{"enabled": false},
			},
		},
	}
}
