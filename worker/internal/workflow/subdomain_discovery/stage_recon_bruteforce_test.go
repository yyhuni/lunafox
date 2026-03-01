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
		ctx:         context.Background(),
		domains:     []string{"example.com"},
		typedConfig: WorkflowConfig{Recon: ReconStageConfig{Enabled: false}},
		workDir:     t.TempDir(),
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
		ctx:         context.Background(),
		domains:     []string{"example.com"},
		typedConfig: WorkflowConfig{Recon: ReconStageConfig{Enabled: true}},
		workDir:     t.TempDir(),
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
		typedConfig: WorkflowConfig{
			Recon: ReconStageConfig{
				Enabled: true,
				Tools: ReconTools{
					Subfinder: ReconSubfinderToolConfig{
						Enabled: true,
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
	toolConfig := ReconSubfinderToolConfig{
		Enabled:        true,
		TimeoutRuntime: 10,
		ThreadsCLI:     5,
	}

	cmd := w.createReconCommand(ctx, "example.com", toolConfig)
	require.NotNil(t, cmd)
	assert.NotContains(t, cmd.Args, "-pc")
}

func TestRunBruteforceStageMissingWordlist(t *testing.T) {
	withNopLogger(t)
	w := &Workflow{runner: activity.NewRunner(t.TempDir())}
	ctx := &workflowContext{
		ctx:     context.Background(),
		domains: []string{"example.com"},
		typedConfig: WorkflowConfig{
			Bruteforce: BruteforceStageConfig{
				Enabled: true,
				Tools: BruteforceTools{
					SubdomainBruteforce: BruteforceSubdomainBruteforceToolConfig{Enabled: true},
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
		typedConfig: WorkflowConfig{
			Bruteforce: BruteforceStageConfig{
				Enabled: true,
				Tools: BruteforceTools{
					SubdomainBruteforce: BruteforceSubdomainBruteforceToolConfig{Enabled: true},
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
		typedConfig: WorkflowConfig{
			Bruteforce: BruteforceStageConfig{
				Enabled: true,
				Tools: BruteforceTools{
					SubdomainBruteforce: BruteforceSubdomainBruteforceToolConfig{
						Enabled:                      true,
						TimeoutRuntime:               10,
						ThreadsCLI:                   10,
						RateLimitCLI:                 10,
						SubdomainWordlistNameRuntime: "wordlist.txt",
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

	raw := BruteforceSubdomainBruteforceToolConfig{
		Enabled:                      true,
		TimeoutRuntime:               10,
		ThreadsCLI:                   10,
		RateLimitCLI:                 10,
		WildcardTestsCLI:             5,
		WildcardBatchCLI:             10,
		SubdomainWordlistNameRuntime: "wordlist.txt",
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
		typedConfig: WorkflowConfig{
			Recon: ReconStageConfig{Enabled: false},
			Bruteforce: BruteforceStageConfig{
				Enabled: true,
				Tools: BruteforceTools{
					SubdomainBruteforce: BruteforceSubdomainBruteforceToolConfig{Enabled: true},
				},
			},
		},
		workDir: t.TempDir(),
	}

	result := w.runAllStages(ctx)
	require.Len(t, result.failed, 1)
	assert.Contains(t, result.failed[0], "missing subdomain-wordlist-name")
}
