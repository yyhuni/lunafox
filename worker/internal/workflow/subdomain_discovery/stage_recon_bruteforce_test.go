package subdomain_discovery

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/yyhuni/lunafox/worker/internal/server"
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
	assert.NotContains(t, cmd.Args, "-pc")
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
	assert.Contains(t, result.failed[0], "missing subdomain-wordlist-name")
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
	cmd := w.createBruteforceCommand(ctx, "exa mple.com", raw, "/tmp/wordlist.txt", "/tmp/resolvers.txt")
	require.NotNil(t, cmd)
	assert.Contains(t, cmd.Args, "/tmp/wordlist.txt")
	assert.Contains(t, cmd.Args, "/tmp/resolvers.txt")
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
