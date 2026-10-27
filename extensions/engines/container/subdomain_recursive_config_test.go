package containercontract_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	enginecontract "github.com/yyhuni/lunafox/contracts/enginemanifest"
)

func TestSubdomainRecursiveConfigUsesManifestDefaultsAndBounds(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "subdomain_discovery", "engine.json"))
	require.NoError(t, err)
	manifest, err := enginecontract.DecodeEngineDefinition(payload, "subdomain_discovery/engine.json")
	require.NoError(t, err)

	defaults, err := engineexecution.NormalizeAndValidateConfig(map[string]any{
		"bruteforce": map[string]any{"enabled": true},
	}, manifest.Execution)
	require.NoError(t, err)
	bruteforce := defaults["bruteforce"].(map[string]any)
	require.Equal(t, false, bruteforce["recursive"])
	require.Equal(t, "1", fmt.Sprint(bruteforce["recursive-depth"]))

	for _, recursive := range []bool{false, true} {
		for _, depth := range []int{-1, 0, 1, 3, 5, 6} {
			t.Run(fmt.Sprintf("recursive=%t/depth=%d", recursive, depth), func(t *testing.T) {
				raw := map[string]any{"bruteforce": map[string]any{
					"enabled": true, "recursive": recursive, "recursive-depth": depth,
				}}
				config, err := engineexecution.NormalizeAndValidateConfig(raw, manifest.Execution)
				if depth < 1 || depth > 5 {
					require.ErrorContains(t, err, "bruteforce.recursive-depth")
					return
				}
				require.NoError(t, err)
				got := config["bruteforce"].(map[string]any)
				require.Equal(t, recursive, got["recursive"])
				require.Equal(t, fmt.Sprint(depth), fmt.Sprint(got["recursive-depth"]))
			})
		}
	}
}
