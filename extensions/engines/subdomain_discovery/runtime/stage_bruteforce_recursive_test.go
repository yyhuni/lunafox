package subdomaindiscoveryruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	subdomainspec "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
)

func TestRunBruteforceStageRecursesByDeterministicLayer(t *testing.T) {
	executor := &recursiveBruteforceExecutor{
		outputs: map[string]string{
			"example.com":      "B.example.com.\na.example.com\na.example.com\ninvalid line\n",
			"a.example.com":    "V1.a.example.com.\nv1.a.example.com\n",
			"b.example.com":    "v1.b.example.com\n",
			"v1.a.example.com": "v2.v1.a.example.com\n",
			"v1.b.example.com": "v2.v1.b.example.com\n",
		},
	}
	run := newStageTestRun(t, true, &recordingStageProgress{}, executor)
	run.typedConfig.Bruteforce.Recursive = true
	run.typedConfig.Bruteforce.RecursiveDepth = 2

	outcome := (&Runtime{}).runBruteforceStage(context.Background(), run)

	require.Empty(t, outcome.failed)
	require.Equal(t, []string{
		"example.com",
		"a.example.com",
		"b.example.com",
		"v1.a.example.com",
		"v1.b.example.com",
	}, executor.domains())
	require.Len(t, outcome.outputArtifactPaths, 5)
	for _, path := range outcome.outputArtifactPaths {
		require.Equal(t, run.workspaceDir, filepath.Dir(path))
		require.FileExists(t, path)
	}
}

func TestRunBruteforceStageSupportsMaximumRecursiveDepth(t *testing.T) {
	outputs := map[string]string{"example.com": "layer1.example.com\n"}
	for layer := 1; layer < 5; layer++ {
		current := "layer" + formatInt64(int64(layer)) + ".example.com"
		next := "layer" + formatInt64(int64(layer+1)) + ".example.com"
		outputs[current] = next + "\n"
	}
	outputs["layer5.example.com"] = "layer6.example.com\n"
	executor := &recursiveBruteforceExecutor{outputs: outputs}
	run := newStageTestRun(t, true, &recordingStageProgress{}, executor)
	run.typedConfig.Bruteforce.Recursive = true
	run.typedConfig.Bruteforce.RecursiveDepth = 5

	outcome := (&Runtime{}).runBruteforceStage(context.Background(), run)

	require.Empty(t, outcome.failed)
	require.Len(t, executor.domains(), 6)
	require.Equal(t, "layer5.example.com", executor.domains()[5])
}

func TestRunBruteforceStageKeepsBaselineWhenRecursionDisabled(t *testing.T) {
	executor := &recursiveBruteforceExecutor{outputs: map[string]string{"example.com": "api.example.com\n"}}
	run := newStageTestRun(t, true, &recordingStageProgress{}, executor)
	run.typedConfig.Bruteforce.Recursive = false
	run.typedConfig.Bruteforce.RecursiveDepth = 1

	outcome := (&Runtime{}).runBruteforceStage(context.Background(), run)

	require.Empty(t, outcome.failed)
	require.Equal(t, []string{"example.com"}, executor.domains())
}

func TestRunBruteforceStageStopsOnEmptyFrontier(t *testing.T) {
	executor := &recursiveBruteforceExecutor{outputs: map[string]string{"example.com": "\ninvalid line\n"}}
	run := newStageTestRun(t, true, &recordingStageProgress{}, executor)
	run.typedConfig.Bruteforce.Recursive = true
	run.typedConfig.Bruteforce.RecursiveDepth = 5

	outcome := (&Runtime{}).runBruteforceStage(context.Background(), run)

	require.Empty(t, outcome.failed)
	require.Equal(t, []string{"example.com"}, executor.domains())
}

func TestRunBruteforceStageFailsFastBeforeLaterParents(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result toolExecutionResult
		err    error
	}{
		{name: "nonzero exit", result: toolExecutionResult{ExitCode: 1}},
		{name: "command error", result: toolExecutionResult{ExitCode: 1, Error: errors.New("puredns failed")}},
		{name: "executor error", err: errors.New("failed to start puredns")},
		{name: "command timeout", result: toolExecutionResult{ExitCode: commandExitCodeTimeout, Error: context.DeadlineExceeded}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			executor := &recursiveBruteforceExecutor{
				outputs:  map[string]string{"example.com": "c.example.com\na.example.com\nb.example.com\n", "a.example.com": "v1.a.example.com\n"},
				results:  map[string]toolExecutionResult{"b.example.com": tc.result},
				failures: map[string]error{"b.example.com": tc.err},
			}
			progress := &recordingStageProgress{}
			run := newStageTestRun(t, true, progress, executor)
			run.typedConfig.Bruteforce.Recursive = true
			run.typedConfig.Bruteforce.RecursiveDepth = 2

			outcome := (&Runtime{}).runAllStages(context.Background(), run)

			require.NotEmpty(t, outcome.failed)
			require.Equal(t, []string{"example.com", "a.example.com", "b.example.com"}, executor.domains())
			require.True(t, progress.contains("fail stage 2/3 bruteforce"))
			require.Equal(t, 0, countProgressMessages(progress, "complete stage 2/3 bruteforce"))
			require.Equal(t, "resolve", executor.invocations[len(executor.invocations)-1].Args[0])
			require.NotEmpty(t, outcome.resultArtifactPath)
			err := stageOutcomeError(context.Background(), outcome)
			require.Error(t, err)
			if errors.Is(tc.result.Error, context.DeadlineExceeded) {
				require.ErrorIs(t, err, context.DeadlineExceeded)
			}
		})
	}
}

func TestRunBruteforceStagePropagatesCancellationBetweenParents(t *testing.T) {
	for _, stopAfter := range []string{"example.com", "a.example.com"} {
		t.Run(stopAfter, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			executor := &recursiveBruteforceExecutor{
				outputs: map[string]string{
					"example.com":   "a.example.com\nb.example.com\n",
					"a.example.com": "v1.a.example.com\n",
				},
				afterCommand: func(domain string) {
					if domain == stopAfter {
						cancel()
					}
				},
			}
			run := newStageTestRun(t, true, &recordingStageProgress{}, executor)
			run.typedConfig.Bruteforce.Recursive = true
			run.typedConfig.Bruteforce.RecursiveDepth = 2

			outcome := (&Runtime{}).runAllStages(ctx, run)

			require.NotEmpty(t, outcome.failed)
			require.ErrorIs(t, outcome.failureCause, context.Canceled)
			require.ErrorIs(t, stageOutcomeError(ctx, outcome), context.Canceled)
			want := []string{"example.com"}
			if stopAfter != "example.com" {
				want = append(want, "a.example.com")
			}
			require.Equal(t, want, executor.domains())
			require.Equal(t, "bruteforce", executor.invocations[len(executor.invocations)-1].Args[0])
		})
	}
}

func TestRunBruteforceStageRejectsRecursiveDepthOutsideManifestBounds(t *testing.T) {
	for _, depth := range []int64{0, 6} {
		t.Run("depth-"+formatInt64(depth), func(t *testing.T) {
			executor := &recursiveBruteforceExecutor{outputs: map[string]string{"example.com": "api.example.com\n"}}
			run := newStageTestRun(t, true, &recordingStageProgress{}, executor)
			run.typedConfig.Bruteforce.RecursiveDepth = depth

			outcome := (&Runtime{}).runBruteforceStage(context.Background(), run)

			require.NotEmpty(t, outcome.failed)
			require.Empty(t, executor.domains())
		})
	}
}

func TestInitializeDiscoveryRunRejectsEnabledInvalidRecursiveDepth(t *testing.T) {
	for _, depth := range []int64{0, 6} {
		t.Run("depth-"+formatInt64(depth), func(t *testing.T) {
			config := subdomainspec.Config{Bruteforce: subdomainspec.BruteforceConfig{Enabled: true, RecursiveDepth: depth}}
			_, err := initializeDiscoveryRun(
				subdomainspec.Target{Type: subdomainspec.TargetTypeDomain, Value: "example.com"},
				config,
				t.TempDir(),
				nil,
				&recursiveBruteforceExecutor{},
				"",
				"",
				"",
				"",
			)
			require.Error(t, err)
		})
	}
}

func TestRunAllStagesResolvesRecursiveArtifactsWithReconUnion(t *testing.T) {
	executor := &recursiveBruteforceExecutor{
		reconOutput: "passive.example.com\n",
		outputs: map[string]string{
			"example.com":     "api.example.com\n",
			"api.example.com": "deep.api.example.com\n",
		},
		resolveOutput: "passive.example.com\napi.example.com\ndeep.api.example.com\n",
	}
	run := newStageTestRun(t, true, &recordingStageProgress{}, executor)
	run.typedConfig.Bruteforce.Recursive = true
	run.typedConfig.Bruteforce.RecursiveDepth = 1

	outcome := (&Runtime{}).runAllStages(context.Background(), run)

	require.Empty(t, outcome.failed)
	require.Equal(t, []string{"subfinder", "puredns", "puredns", "puredns"}, invocationBinaries(executor.invocations))
	resolveInput, err := os.ReadFile(executor.invocations[3].Args[1])
	require.NoError(t, err)
	require.Equal(t, "api.example.com\ndeep.api.example.com\npassive.example.com\n", string(resolveInput))
}

func TestRecursiveBruteforceReusesConfiguredPureDNSArguments(t *testing.T) {
	executor := &recursiveBruteforceExecutor{outputs: map[string]string{
		"example.com":     "api.example.com\n",
		"api.example.com": "deep.example.com\n",
	}}
	progress := &recordingStageProgress{}
	run := newStageTestRun(t, true, progress, executor)
	run.typedConfig.Bruteforce.Recursive = true
	run.typedConfig.Bruteforce.RecursiveDepth = 1
	run.typedConfig.Bruteforce.Threads = 17
	run.typedConfig.Bruteforce.RateLimit = 23
	run.typedConfig.Bruteforce.Timeout = 47
	run.typedConfig.Bruteforce.WildcardFilter = true
	run.typedConfig.Bruteforce.WildcardProbeCount = 29
	run.typedConfig.Bruteforce.WildcardBatch = 31

	outcome := (&Runtime{}).runBruteforceStage(context.Background(), run)

	require.Empty(t, outcome.failed)
	require.Len(t, executor.invocations, 2)
	require.Equal(t, 2, countProgressMessages(progress, "start command stage=bruteforce"))
	for _, invocation := range executor.invocations {
		require.Equal(t, int64(47), invocation.TimeoutSeconds)
		require.Equal(t, "/wordlist.txt", invocation.Args[1])
		require.Equal(t, "/bruteforce-resolvers.txt", mustCommandArgumentValue(t, invocation.Args, "-r"))
		require.Equal(t, "17", mustCommandArgumentValue(t, invocation.Args, "-t"))
		require.Equal(t, "23", mustCommandArgumentValue(t, invocation.Args, "--rate-limit"))
		require.Equal(t, "29", mustCommandArgumentValue(t, invocation.Args, "--wildcard-tests"))
		require.Equal(t, "31", mustCommandArgumentValue(t, invocation.Args, "--wildcard-batch"))
	}
}

func countProgressMessages(progress *recordingStageProgress, want string) int {
	count := 0
	for _, message := range progress.messages {
		if strings.Contains(message, want) {
			count++
		}
	}
	return count
}

func formatInt64(value int64) string {
	return strconv.FormatInt(value, 10)
}

func mustCommandArgumentValue(t *testing.T, args []string, flag string) string {
	t.Helper()
	value, ok := commandArgumentValue(args, flag)
	require.True(t, ok, "missing %s in %#v", flag, args)
	return value
}

type recursiveBruteforceExecutor struct {
	invocations   []toolInvocation
	reconOutput   string
	outputs       map[string]string
	resolveOutput string
	failures      map[string]error
	results       map[string]toolExecutionResult
	afterCommand  func(string)
}

func (executor *recursiveBruteforceExecutor) execute(ctx context.Context, command toolInvocation) (toolExecutionResult, error) {
	if err := ctx.Err(); err != nil {
		return toolExecutionResult{}, err
	}
	executor.invocations = append(executor.invocations, toolInvocation{
		Label: command.Label, Binary: command.Binary, Args: append([]string(nil), command.Args...), TimeoutSeconds: command.TimeoutSeconds,
	})
	if command.Binary == "subfinder" {
		outputPath, ok := commandArgumentValue(command.Args, "-o")
		if !ok {
			return toolExecutionResult{}, errors.New("subfinder output path is required")
		}
		return toolExecutionResult{}, os.WriteFile(outputPath, []byte(executor.reconOutput), 0600)
	}
	if command.Binary != "puredns" || len(command.Args) == 0 {
		return toolExecutionResult{}, errors.New("unexpected command")
	}
	outputPath, ok := commandArgumentValue(command.Args, "--write")
	if !ok {
		return toolExecutionResult{}, errors.New("puredns output path is required")
	}
	if command.Args[0] == "resolve" {
		return toolExecutionResult{}, os.WriteFile(outputPath, []byte(executor.resolveOutput), 0600)
	}
	if command.Args[0] != "bruteforce" || len(command.Args) < 3 {
		return toolExecutionResult{}, errors.New("unexpected puredns mode")
	}
	domain := command.Args[2]
	if err := executor.failures[domain]; err != nil {
		return toolExecutionResult{}, err
	}
	if result, ok := executor.results[domain]; ok {
		return result, nil
	}
	if err := os.WriteFile(outputPath, []byte(executor.outputs[domain]), 0600); err != nil {
		return toolExecutionResult{}, err
	}
	if executor.afterCommand != nil {
		executor.afterCommand(domain)
	}
	return toolExecutionResult{}, nil
}

func (executor *recursiveBruteforceExecutor) domains() []string {
	domains := make([]string, 0)
	for _, invocation := range executor.invocations {
		if invocation.Binary == "puredns" && len(invocation.Args) > 2 && invocation.Args[0] == "bruteforce" {
			domains = append(domains, invocation.Args[2])
		}
	}
	return domains
}

var _ toolExecutor = (*recursiveBruteforceExecutor)(nil)
