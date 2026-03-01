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

func workerRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
