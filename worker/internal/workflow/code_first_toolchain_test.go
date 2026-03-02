package workflow_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodeFirstToolchain_MigratedFromTemplateGenerators(t *testing.T) {
	root := workerRoot(t)

	keptDirs := []string{
		filepath.Join(root, "cmd", "workflow-contract-gen"),
	}
	for _, dir := range keptDirs {
		info, err := os.Stat(dir)
		require.NoErrorf(t, err, "expected code-first generator directory to exist: %s", dir)
		require.Truef(t, info.IsDir(), "expected directory path: %s", dir)
	}

	removedDirs := []string{
		filepath.Join(root, "cmd", "const-gen"),
		filepath.Join(root, "cmd", "doc-gen"),
		filepath.Join(root, "cmd", "schema-gen"),
	}
	for _, dir := range removedDirs {
		_, err := os.Stat(dir)
		require.Truef(
			t,
			errors.Is(err, os.ErrNotExist),
			"legacy template generator must be removed for code-first migration: %s",
			dir,
		)
	}
}

func TestCodeFirstToolchain_HasAuthoritativeGlobalGenerationEntry(t *testing.T) {
	root := workerRoot(t)
	makefilePath := filepath.Join(root, "Makefile")
	content, err := os.ReadFile(makefilePath)
	require.NoError(t, err)
	text := string(content)

	require.Contains(t, text, "workflow-contracts-gen-all:")
	require.Contains(t, text, "generate: workflow-contracts-gen-all")
}

func TestCodeFirstToolchain_MetadataAndCleanUseDynamicWorkflowDiscovery(t *testing.T) {
	root := workerRoot(t)
	makefilePath := filepath.Join(root, "Makefile")
	content, err := os.ReadFile(makefilePath)
	require.NoError(t, err)
	text := string(content)

	require.Contains(t, text, "for workflow_dir in internal/workflow/*/")
	require.NotContains(t, text, "go test -v ./internal/workflow/subdomain_discovery")
	require.NotContains(t, text, "schema_generated.json")
}

func TestCodeFirstToolchain_NoPerWorkflowContractAssetsBoilerplate(t *testing.T) {
	root := workerRoot(t)
	workflowRoot := filepath.Join(root, "internal", "workflow")
	entries, err := os.ReadDir(workflowRoot)
	require.NoError(t, err)

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "all" {
			continue
		}
		workflowDir := filepath.Join(workflowRoot, entry.Name())
		if _, err := os.Stat(filepath.Join(workflowDir, "contract_definition.go")); err != nil {
			continue
		}
		contractAssets := filepath.Join(workflowDir, "contract_assets.go")
		_, err := os.Stat(contractAssets)
		require.Truef(
			t,
			errors.Is(err, os.ErrNotExist),
			"workflow %s should not keep per-workflow contract_assets.go boilerplate",
			entry.Name(),
		)
	}
}

func workerRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
