package subdomain_discovery

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/yyhuni/lunafox/worker/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubServerClient struct {
	err error
}

func (s stubServerClient) GetProviderConfig(ctx context.Context, scanID int, toolName string) (*server.ProviderConfig, error) {
	return nil, nil
}

func (s stubServerClient) EnsureWordlistLocal(ctx context.Context, wordlistName, basePath string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return "/tmp/wordlist.txt", nil
}

func (s stubServerClient) PostBatch(ctx context.Context, scanID, targetID int, dataType string, items []any) error {
	return nil
}

func TestRunReconStageDisabled(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{runner: activity.NewRunner(t.TempDir())}
	ctx := &workflowContext{
		ctx:     context.Background(),
		domains: []string{"example.com"},
		config: map[string]any{
			stageRecon: map[string]any{
				"enabled": false,
				"tools": map[string]any{
					toolSubfinder: map[string]any{"enabled": true},
				},
			},
		},
		workDir: t.TempDir(),
	}

	result := w.runReconStage(ctx)
	assert.Empty(t, result.files)
	assert.Empty(t, result.failed)
	assert.Empty(t, result.success)
}

func TestRunReconStageMissingTools(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{runner: activity.NewRunner(t.TempDir())}
	ctx := &workflowContext{
		ctx:     context.Background(),
		domains: []string{"example.com"},
		config: map[string]any{
			stageRecon: map[string]any{
				"enabled": true,
			},
		},
		workDir: t.TempDir(),
	}

	result := w.runReconStage(ctx)
	assert.Empty(t, result.files)
}

func TestRunReconStageMissingConfigForTool(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{runner: activity.NewRunner(t.TempDir())}
	ctx := &workflowContext{
		ctx:     context.Background(),
		domains: []string{"example.com"},
		config: map[string]any{
			stageRecon: map[string]any{
				"enabled": true,
				"tools": map[string]any{
					toolSubfinder: map[string]any{
						"enabled": true,
					},
				},
			},
		},
		workDir: t.TempDir(),
	}

	result := w.runReconStage(ctx)
	assert.Empty(t, result.files)
	assert.Empty(t, result.failed)
}

func TestCreateReconCommandNoProviderConfig(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{}
	ctx := &workflowContext{
		workDir: t.TempDir(),
	}
	toolConfig := map[string]any{
		"timeout-runtime": 10,
		"threads-cli":     5,
	}

	cmd := w.createReconCommand(ctx, "example.com", toolSubfinder, toolConfig)
	require.NotNil(t, cmd)
	assert.NotContains(t, cmd.Command, "-pc")
}

func TestRunBruteforceStageMissingWordlist(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{runner: activity.NewRunner(t.TempDir())}
	ctx := &workflowContext{
		ctx:     context.Background(),
		domains: []string{"example.com"},
		config: map[string]any{
			stageBruteforce: map[string]any{
				"tools": map[string]any{
					toolSubdomainBruteforce: map[string]any{
						"enabled": true,
					},
				},
			},
		},
		workDir: t.TempDir(),
	}

	result := w.runBruteforceStage(ctx)
	require.Len(t, result.failed, 1)
	assert.Contains(t, result.failed[0], "missing subdomain-wordlist-name")
}

func TestRunBruteforceStageInvalidConfig(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{runner: activity.NewRunner(t.TempDir())}
	ctx := &workflowContext{
		ctx:     context.Background(),
		domains: []string{"example.com"},
		config: map[string]any{
			stageBruteforce: map[string]any{
				"tools": map[string]any{
					toolSubdomainBruteforce: map[string]any{
						"bad-key": 1,
					},
				},
			},
		},
		workDir: t.TempDir(),
	}

	result := w.runBruteforceStage(ctx)
	require.Len(t, result.failed, 1)
	assert.Contains(t, result.failed[0], "invalid config")
}

func TestRunBruteforceStageWordlistError(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{runner: activity.NewRunner(t.TempDir())}
	ctx := &workflowContext{
		ctx:          context.Background(),
		domains:      []string{"example.com"},
		serverClient: stubServerClient{err: errors.New("boom")},
		config: map[string]any{
			stageBruteforce: map[string]any{
				"tools": map[string]any{
					toolSubdomainBruteforce: map[string]any{
						"timeout-runtime":                 10,
						"threads-cli":                     10,
						"rate-limit-cli":                  10,
						"subdomain-wordlist-name-runtime": "wordlist.txt",
					},
				},
			},
		},
		workDir: t.TempDir(),
	}

	result := w.runBruteforceStage(ctx)
	require.Len(t, result.failed, 1)
	assert.Equal(t, stageBruteforce+" (wordlist: boom)", result.failed[0])
}

func TestCreateBruteforceCommandSuccess(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{}
	ctx := &workflowContext{workDir: t.TempDir()}

	raw := map[string]any{
		"timeout-runtime":                 10,
		"threads-cli":                     10,
		"rate-limit-cli":                  10,
		"wildcard-tests-cli":              5,
		"wildcard-batch-cli":              10,
		"subdomain-wordlist-name-runtime": "wordlist.txt",
	}
	normalized, err := normalizeToolConfig(toolSubdomainBruteforce, raw)
	require.NoError(t, err)

	cmd := w.createBruteforceCommand(ctx, "exa mple.com", normalized, "/tmp/wordlist.txt", "/tmp/resolvers.txt")
	if cmd == nil {
		params := map[string]any{
			"Domain":     "exa mple.com",
			"OutputFile": filepath.Join(ctx.workDir, "bruteforce_exa_mple.com.txt"),
			"Wordlist":   "/tmp/wordlist.txt",
			"Resolvers":  "/tmp/resolvers.txt",
		}
		cmdStr, err := buildCommand(toolSubdomainBruteforce, params, normalized)
		t.Fatalf("expected command, build err=%v, cmd=%s", err, cmdStr)
	}
	require.NotNil(t, cmd)
	assert.Contains(t, cmd.Command, "/tmp/wordlist.txt")
	assert.Contains(t, cmd.Command, "/tmp/resolvers.txt")
}

func TestRunAllStagesBruteforceFailure(t *testing.T) {
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
				"enabled": true,
				"tools": map[string]any{
					toolSubdomainBruteforce: map[string]any{},
				},
			},
		},
		workDir: t.TempDir(),
	}

	result := w.runAllStages(ctx)
	require.Len(t, result.failed, 1)
	assert.Contains(t, result.failed[0], "missing subdomain-wordlist-name")
}
