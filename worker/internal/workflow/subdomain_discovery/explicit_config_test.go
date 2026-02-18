package subdomain_discovery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateExplicitConfig_RequiresStageConfig(t *testing.T) {
	err := validateExplicitConfig(map[string]any{})
	require.Error(t, err)
}

func TestValidateExplicitConfig_RequiresEnabledFlags(t *testing.T) {
	config := map[string]any{
		stageRecon: map[string]any{
			"tools": map[string]any{},
		},
	}
	err := validateExplicitConfig(config)
	require.Error(t, err)
}

func TestValidateExplicitConfig_AllStagesAndToolsPresent(t *testing.T) {
	config := map[string]any{
		stageRecon: map[string]any{
			"enabled": true,
			"tools": map[string]any{
				toolSubfinder: map[string]any{
					"enabled":         true,
					"timeout-runtime": 3600,
					"threads-cli":     10,
				},
				toolAssetfinder: map[string]any{
					"enabled":         true,
					"timeout-runtime": 3600,
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

	require.NoError(t, validateExplicitConfig(config))
}

func TestValidateExplicitConfig_NilConfig(t *testing.T) {
	require.Error(t, validateExplicitConfig(nil))
}
