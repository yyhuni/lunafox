package subdomaindiscoveryruntime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveInputArtifactPathsNormalizesRelativePaths(t *testing.T) {
	workspace := t.TempDir()
	absoluteInput := filepath.Join(workspace, "absolute.txt")

	paths, err := resolveInputArtifactPaths(workspace, []string{" relative.txt ", absoluteInput, ""})
	require.NoError(t, err)
	require.Equal(t, []string{
		filepath.Join(workspace, "relative.txt"),
		absoluteInput,
	}, paths)
}

func TestDeduplicateSubdomainArtifactsReturnsOutputPath(t *testing.T) {
	workspace := t.TempDir()
	input := filepath.Join(workspace, "input.txt")
	require.NoError(t, os.WriteFile(input, []byte("b.example.com\na.example.com\na.example.com\n"), 0600))

	outputPath, err := deduplicateSubdomainArtifactsWithChunkSize(nil, workspace, []string{input}, 2)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(workspace, "deduplicated_subdomains_001.txt"), outputPath)

	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	require.Equal(t, "a.example.com\nb.example.com\n", string(content))
}

func TestRequireWorkspaceAbsRequiresExistingDirectory(t *testing.T) {
	missingWorkspace := filepath.Join(t.TempDir(), "missing-workspace")

	_, err := requireWorkspaceAbs(missingWorkspace, "execution workspace")
	require.Error(t, err)
	require.Contains(t, err.Error(), "execution workspace must be an existing directory")
}

func TestRequireWorkspaceAbsRejectsRelativeWorkspace(t *testing.T) {
	relativeWorkspace := "relative-workspace"
	require.NoError(t, os.Mkdir(relativeWorkspace, 0700))
	t.Cleanup(func() { _ = os.Remove(relativeWorkspace) })

	_, err := requireWorkspaceAbs(relativeWorkspace, "execution workspace")
	require.Error(t, err)
	require.Contains(t, err.Error(), "execution workspace must be absolute")
}

func TestRequireWorkspaceAbsRejectsNonCleanWorkspace(t *testing.T) {
	workspace := t.TempDir()
	nonCleanWorkspace := workspace + string(filepath.Separator) + "."

	_, err := requireWorkspaceAbs(nonCleanWorkspace, "execution workspace")
	require.Error(t, err)
	require.Contains(t, err.Error(), "execution workspace must be clean")
}

func TestRequireWorkspaceAbsRejectsSurroundingWhitespace(t *testing.T) {
	workspace := t.TempDir()

	_, err := requireWorkspaceAbs(" "+workspace+" ", "execution workspace")
	require.Error(t, err)
	require.Contains(t, err.Error(), "execution workspace must not have surrounding whitespace")
}

func TestRequireWorkspaceAbsRejectsFile(t *testing.T) {
	workspaceFile := filepath.Join(t.TempDir(), "workspace-file")
	require.NoError(t, os.WriteFile(workspaceFile, []byte("not a directory"), 0600))

	_, err := requireWorkspaceAbs(workspaceFile, "execution workspace")
	require.Error(t, err)
	require.Contains(t, err.Error(), "execution workspace must be an existing directory")
}
