package subdomain_discovery

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodeFirstMigration_NoLegacyTemplateArtifacts(t *testing.T) {
	legacyFiles := []string{
		"templates.yaml",
		"template_loader.go",
		"constants_generated.go",
		"template_parsing_test.go",
		"metadata_test.go",
	}

	for _, legacyFile := range legacyFiles {
		_, err := os.Stat(legacyFile)
		require.Truef(
			t,
			errors.Is(err, os.ErrNotExist),
			"legacy template artifact must be removed for code-first migration: %s",
			legacyFile,
		)
	}
}

func TestCodeFirstMigration_NoTinySplitRuntimeFiles(t *testing.T) {
	runtimeSplitFiles := []string{
		"stage_runner.go",
		"subdomain_file_runtime.go",
		"config_decode_utils.go",
		"contract_assets.go",
	}

	for _, runtimeSplitFile := range runtimeSplitFiles {
		_, err := os.Stat(runtimeSplitFile)
		require.Truef(
			t,
			errors.Is(err, os.ErrNotExist),
			"tiny split runtime file should be merged to improve readability: %s",
			runtimeSplitFile,
		)
	}
}
