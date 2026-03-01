package subdomain_discovery

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"go.uber.org/zap"
)

func withNopLogger(t *testing.T) {
	t.Helper()
	prev := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() {
		pkg.Logger = prev
	})
}

func TestBuildWildcardSettings(t *testing.T) {
	withNopLogger(t)
	cfg := PermutationSubdomainPermutationResolveToolConfig{
		Enabled:                           true,
		TimeoutRuntime:                    5,
		WildcardSampleTimeoutRuntime:      5,
		WildcardTestsCLI:                  3,
		WildcardBatchCLI:                  4,
		WildcardSampleMultiplierRuntime:   2,
		WildcardExpansionThresholdRuntime: 10,
	}

	settings, err := buildWildcardSettings(cfg, "/tmp/resolvers.txt")
	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, settings.sampleTimeout)
	assert.Equal(t, 3, settings.tests)
	assert.Equal(t, 4, settings.batch)
	assert.Equal(t, 2, settings.sampleMultiplier)
	assert.Equal(t, 10, settings.expansionThreshold)
	assert.Equal(t, "/tmp/resolvers.txt", settings.resolversPath)
}

func TestBuildWildcardSettingsInvalid(t *testing.T) {
	withNopLogger(t)
	cfg := PermutationSubdomainPermutationResolveToolConfig{
		Enabled:                           true,
		TimeoutRuntime:                    1,
		WildcardSampleTimeoutRuntime:      0,
		WildcardTestsCLI:                  1,
		WildcardBatchCLI:                  1,
		WildcardSampleMultiplierRuntime:   1,
		WildcardExpansionThresholdRuntime: 1,
	}

	_, err := buildWildcardSettings(cfg, "/tmp/resolvers.txt")
	require.Error(t, err)

	cfg.WildcardSampleTimeoutRuntime = 1
	_, err = buildWildcardSettings(cfg, "")
	require.Error(t, err)
}

func TestMergeFilesDedupAndFilter(t *testing.T) {
	withNopLogger(t)
	dir := t.TempDir()
	file1 := filepath.Join(dir, "one.txt")
	file2 := filepath.Join(dir, "two.txt")

	require.NoError(t, os.WriteFile(file1, []byte(strings.Join([]string{
		"A.example.com",
		"# comment",
		"bad domain",
		"127.0.0.1",
		"b.example.com",
		"",
	}, "\n")), 0644))
	require.NoError(t, os.WriteFile(file2, []byte("a.example.com\nc.example.com\n"), 0644))

	output := filepath.Join(dir, "merged.txt")
	w := &Workflow{}
	require.NoError(t, w.mergeFiles([]string{file1, file2}, output))

	lines := readLines(t, output)
	assert.Equal(t, []string{"A.example.com", "b.example.com", "c.example.com"}, lines)
}

func TestStreamMergeFileMissingFile(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{}
	buf := &strings.Builder{}
	writer := bufio.NewWriter(buf)
	require.NoError(t, w.streamMergeFile(filepath.Join(t.TempDir(), "missing.txt"), map[string]struct{}{}, writer))
}

func TestCheckWildcardEmptyFile(t *testing.T) {
	withNopLogger(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.txt")
	require.NoError(t, os.WriteFile(input, []byte(""), 0644))

	w := &Workflow{}
	result := w.checkWildcard(context.Background(), input, dir, wildcardSettings{})
	assert.False(t, result.isWildcard)
	assert.Equal(t, "empty input file", result.reason)
}

func TestRunMergeStageMissingConfig(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{runner: activity.NewRunner(t.TempDir())}
	ctx := &workflowContext{
		ctx: context.Background(),
		typedConfig: WorkflowConfig{
			Resolve: ResolveStageConfig{
				Enabled: true,
				Tools: ResolveTools{
					SubdomainResolve: ResolveSubdomainResolveToolConfig{Enabled: true},
				},
			},
		},
		workDir: t.TempDir(),
	}

	result := w.runMergeStage(ctx, []string{}, stageResolve, toolSubdomainResolve)
	assert.Equal(t, []string{stageResolve}, result.failed)
	assert.Empty(t, result.files)
}

func TestRunMergeStageMissingTools(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{runner: activity.NewRunner(t.TempDir())}
	ctx := &workflowContext{
		ctx: context.Background(),
		typedConfig: WorkflowConfig{
			Resolve: ResolveStageConfig{
				Enabled: true,
			},
		},
		workDir: t.TempDir(),
	}

	result := w.runMergeStage(ctx, []string{}, stageResolve, toolSubdomainResolve)
	assert.Equal(t, []string{stageResolve}, result.failed)
}

func TestRunMergeStageUnknownTool(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{runner: activity.NewRunner(t.TempDir())}
	ctx := &workflowContext{
		ctx: context.Background(),
		typedConfig: WorkflowConfig{
			Resolve: ResolveStageConfig{
				Enabled: true,
				Tools: ResolveTools{
					SubdomainResolve: ResolveSubdomainResolveToolConfig{
						Enabled:        true,
						TimeoutRuntime: 10,
						ThreadsCLI:     10,
						RateLimitCLI:   10,
					},
				},
			},
		},
		workDir: t.TempDir(),
	}

	result := w.runMergeStage(ctx, []string{}, stageResolve, "missing")
	assert.Equal(t, []string{stageResolve}, result.failed)
}

func TestRunMergeStageMergeFilesError(t *testing.T) {
	withNopLogger(t)
	baseDir := t.TempDir()
	invalidDir := filepath.Join(baseDir, "missing", "dir")

	w := &Workflow{runner: activity.NewRunner(baseDir)}
	ctx := &workflowContext{
		ctx: context.Background(),
		typedConfig: WorkflowConfig{
			Resolve: ResolveStageConfig{
				Enabled: true,
				Tools: ResolveTools{
					SubdomainResolve: ResolveSubdomainResolveToolConfig{
						Enabled:        true,
						TimeoutRuntime: 10,
						ThreadsCLI:     10,
						RateLimitCLI:   10,
					},
				},
			},
		},
		workDir: invalidDir,
	}

	result := w.runMergeStage(ctx, []string{"missing.txt"}, stageResolve, toolSubdomainResolve)
	assert.Equal(t, []string{stageResolve}, result.failed)
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	file, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	require.NoError(t, scanner.Err())
	return lines
}
