package subdomain_discovery

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStageResultMerge(t *testing.T) {
	left := stageResult{
		files:   []string{"a.txt"},
		failed:  []string{"tool-a"},
		success: []string{"tool-b"},
	}
	right := stageResult{
		files:   []string{"b.txt"},
		failed:  []string{"tool-c"},
		success: []string{"tool-d"},
	}

	left.merge(right)
	assert.Equal(t, []string{"a.txt", "b.txt"}, left.files)
	assert.Equal(t, []string{"tool-a", "tool-c"}, left.failed)
	assert.Equal(t, []string{"tool-b", "tool-d"}, left.success)
}

func TestProcessResults(t *testing.T) {
	results := []*activity.Result{
		{Name: "ok", OutputFile: "ok.txt"},
		{Name: "bad", Error: errors.New("fail")},
	}

	out := processResults(results)
	assert.Equal(t, []string{"ok.txt"}, out.files)
	assert.Equal(t, []string{"ok"}, out.success)
	assert.Equal(t, []string{"bad"}, out.failed)
}

func TestCreateReconCommandSuccess(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{}
	ctx := &workflowContext{
		workDir:            t.TempDir(),
		providerConfigPath: "/tmp/provider.conf",
	}
	toolConfig := map[string]any{
		"timeout-runtime": 10,
		"threads-cli":     5,
	}

	cmd := w.createReconCommand(ctx, "exa mple.com", toolSubfinder, toolConfig)
	require.NotNil(t, cmd)
	assert.Equal(t, 10*time.Second, cmd.Timeout)
	assert.Contains(t, cmd.Command, "-pc")
	assert.Contains(t, cmd.Command, "/tmp/provider.conf")
	assert.True(t, strings.Contains(filepath.Base(cmd.OutputFile), "exa_mple.com"))
	assert.True(t, strings.Contains(filepath.Base(cmd.LogFile), "exa_mple.com"))
}

func TestCreateReconCommandMissingConfig(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{}
	ctx := &workflowContext{
		workDir: t.TempDir(),
	}
	cmd := w.createReconCommand(ctx, "example.com", toolSubfinder, map[string]any{})
	assert.Nil(t, cmd)
}

func TestRunAllStagesNoMergeWithoutFiles(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{runner: activity.NewRunner(t.TempDir())}
	ctx := &workflowContext{
		ctx:     context.Background(),
		domains: []string{"example.com"},
		config: map[string]any{
			stageRecon: map[string]any{
				"enabled": false,
			},
			stageBruteforce: map[string]any{
				"enabled": false,
			},
			stagePermutation: map[string]any{
				"enabled": true,
				"tools": map[string]any{
					toolSubdomainPermutationResolve: map[string]any{"enabled": true},
				},
			},
			stageResolve: map[string]any{
				"enabled": true,
				"tools": map[string]any{
					toolSubdomainResolve: map[string]any{"enabled": true},
				},
			},
		},
		workDir: t.TempDir(),
	}

	result := w.runAllStages(ctx)
	assert.Empty(t, result.files)
	assert.Empty(t, result.failed)
	assert.Empty(t, result.success)
}
