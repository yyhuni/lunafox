package subdomaindiscoveryruntime

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	subdomainspec "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
)

func TestRunAllStagesResolvesDeduplicatedCandidateUnionOnce(t *testing.T) {
	progress := &recordingStageProgress{}
	executor := &recordingStageExecutor{
		reconOutput:      "API.example.com.\nshared.example.com\ninvalid line\n",
		bruteforceOutput: "shared.example.com.\nWWW.example.com\napi.example.com\n# comment\n",
		resolveOutput:    "api.example.com\nshared.example.com\nwww.example.com\n",
	}
	run := newStageTestRun(t, true, progress, executor)

	outcome := (&Runtime{}).runAllStages(context.Background(), run)

	require.Empty(t, outcome.failed)
	require.NotEmpty(t, outcome.resultArtifactPath)
	require.Equal(t, []string{"subfinder", "puredns", "puredns"}, invocationBinaries(executor.invocations))
	require.Equal(t, "bruteforce", executor.invocations[1].Args[0])
	require.Equal(t, "resolve", executor.invocations[2].Args[0])
	bruteforceResolvers, ok := commandArgumentValue(executor.invocations[1].Args, "-r")
	require.True(t, ok)
	require.Equal(t, "/bruteforce-resolvers.txt", bruteforceResolvers)
	resolveResolvers, ok := commandArgumentValue(executor.invocations[2].Args, "-r")
	require.True(t, ok)
	require.Equal(t, "/resolve-resolvers.txt", resolveResolvers)
	resolveOutputPath, ok := commandArgumentValue(executor.invocations[2].Args, "--write")
	require.True(t, ok)
	require.Equal(t, resolveOutputPath, outcome.resultArtifactPath)

	resolveInput, err := os.ReadFile(executor.invocations[2].Args[1])
	require.NoError(t, err)
	require.Equal(t, "api.example.com\nshared.example.com\nwww.example.com\n", string(resolveInput))

	for _, invocation := range executor.invocations {
		require.NotEqual(t, "dnsgen", invocation.Binary)
		if invocation.Binary == "puredns" {
			require.True(t, hasArgumentPrefix(invocation.Args, "--skip-wildcard-filter"))
		}
	}
	require.NotContains(t, executor.invocations[1].Args, "--wildcard-tests")
	require.NotContains(t, executor.invocations[1].Args, "--wildcard-batch")
	require.True(t, progress.contains("start stage 1/3 recon"))
	require.True(t, progress.contains("start stage 2/3 bruteforce"))
	require.True(t, progress.contains("start stage 3/3 resolve"))
}

func TestRunAllStagesAppliesWildcardFilterPerStage(t *testing.T) {
	progress := &recordingStageProgress{}
	executor := &recordingStageExecutor{
		reconOutput:      "api.example.com\n",
		bruteforceOutput: "admin.example.com\n",
		resolveOutput:    "api.example.com\nadmin.example.com\n",
	}
	run := newStageTestRun(t, true, progress, executor)
	run.typedConfig.Bruteforce.WildcardFilter = true
	run.typedConfig.Resolve.WildcardFilter = false

	outcome := (&Runtime{}).runAllStages(context.Background(), run)

	require.Empty(t, outcome.failed)
	require.Len(t, executor.invocations, 3)
	bruteforce := executor.invocations[1].Args
	resolve := executor.invocations[2].Args
	require.False(t, hasArgumentPrefix(bruteforce, "--skip-wildcard-filter"))
	require.Contains(t, bruteforce, "--wildcard-tests")
	require.Contains(t, bruteforce, "--wildcard-batch")
	require.True(t, hasArgumentPrefix(resolve, "--skip-wildcard-filter"))
}

func TestRunAllStagesCanEnableResolveWildcardFilterWithoutBruteforce(t *testing.T) {
	progress := &recordingStageProgress{}
	executor := &recordingStageExecutor{
		reconOutput:   "api.example.com\n",
		resolveOutput: "api.example.com\n",
	}
	run := newStageTestRun(t, false, progress, executor)
	run.typedConfig.Resolve.WildcardFilter = true

	outcome := (&Runtime{}).runAllStages(context.Background(), run)

	require.Empty(t, outcome.failed)
	require.Len(t, executor.invocations, 2)
	require.Equal(t, "resolve", executor.invocations[1].Args[0])
	require.False(t, hasArgumentPrefix(executor.invocations[1].Args, "--skip-wildcard-filter"))
}

func TestRunAllStagesSkipsDisabledBruteforce(t *testing.T) {
	progress := &recordingStageProgress{}
	executor := &recordingStageExecutor{
		reconOutput:   "Portal.example.com.\n",
		resolveOutput: "portal.example.com\n",
	}
	run := newStageTestRun(t, false, progress, executor)

	outcome := (&Runtime{}).runAllStages(context.Background(), run)

	require.Empty(t, outcome.failed)
	require.NotEmpty(t, outcome.resultArtifactPath)
	require.Equal(t, []string{"subfinder", "puredns"}, invocationBinaries(executor.invocations))
	require.Equal(t, "resolve", executor.invocations[1].Args[0])
	resolveInput, err := os.ReadFile(executor.invocations[1].Args[1])
	require.NoError(t, err)
	require.Equal(t, "portal.example.com\n", string(resolveInput))
}

func TestRunAllStagesSkipsResolveWhenUpstreamArtifactsContainNoCandidates(t *testing.T) {
	progress := &recordingStageProgress{}
	executor := &recordingStageExecutor{}
	run := newStageTestRun(t, false, progress, executor)

	outcome := (&Runtime{}).runAllStages(context.Background(), run)

	require.Empty(t, outcome.failed)
	require.Empty(t, outcome.resultArtifactPath)
	require.Equal(t, []string{"subfinder"}, invocationBinaries(executor.invocations))
	require.Contains(t, progress.messages, "skip stage 3/3 resolve reason=no previous outputs")
}

func newStageTestRun(t *testing.T, bruteforceEnabled bool, progress *recordingStageProgress, executor *recordingStageExecutor) *discoveryRun {
	t.Helper()
	return &discoveryRun{
		domain: "example.com",
		typedConfig: subdomainspec.Config{
			Recon: subdomainspec.ReconConfig{
				Enabled: true,
				Timeout: 60,
				Threads: 1,
			},
			Bruteforce: subdomainspec.BruteforceConfig{
				Enabled:            bruteforceEnabled,
				Timeout:            60,
				Resolvers:          "/bruteforce-resolvers.txt",
				Threads:            1,
				RateLimit:          1,
				WildcardFilter:     false,
				WildcardProbeCount: 3,
				WildcardBatch:      100,
			},
			Resolve: subdomainspec.ResolveConfig{
				Enabled:        true,
				Timeout:        60,
				Threads:        1,
				RateLimit:      1,
				WildcardFilter: false,
				Resolvers:      "/resolve-resolvers.txt",
			},
		},
		workspaceDir:            t.TempDir(),
		progress:                progress,
		executor:                executor,
		providerConfigPath:      "/provider.yaml",
		wordlistPath:            "/wordlist.txt",
		bruteforceResolversPath: "/bruteforce-resolvers.txt",
		resolveResolversPath:    "/resolve-resolvers.txt",
	}
}

type recordingStageProgress struct {
	messages []string
}

func (progress *recordingStageProgress) report(message string) error {
	progress.messages = append(progress.messages, message)
	return nil
}

func (progress *recordingStageProgress) contains(want string) bool {
	for _, message := range progress.messages {
		if strings.Contains(message, want) {
			return true
		}
	}
	return false
}

type recordingStageExecutor struct {
	invocations      []toolInvocation
	reconOutput      string
	bruteforceOutput string
	resolveOutput    string
}

func (executor *recordingStageExecutor) execute(_ context.Context, command toolInvocation) (toolExecutionResult, error) {
	executor.invocations = append(executor.invocations, toolInvocation{
		Label:          command.Label,
		Binary:         command.Binary,
		Args:           append([]string(nil), command.Args...),
		TimeoutSeconds: command.TimeoutSeconds,
	})

	outputPath, content, err := executor.outputFor(command)
	if err != nil {
		return toolExecutionResult{}, err
	}
	if err := os.WriteFile(outputPath, []byte(content), 0600); err != nil {
		return toolExecutionResult{}, err
	}
	return toolExecutionResult{ExitCode: 0}, nil
}

func (executor *recordingStageExecutor) outputFor(command toolInvocation) (string, string, error) {
	switch {
	case command.Binary == "subfinder":
		outputPath, ok := commandArgumentValue(command.Args, "-o")
		if !ok {
			return "", "", fmt.Errorf("subfinder output path is required")
		}
		return outputPath, executor.reconOutput, nil
	case command.Binary == "puredns" && len(command.Args) > 0 && command.Args[0] == "bruteforce":
		outputPath, ok := commandArgumentValue(command.Args, "--write")
		if !ok {
			return "", "", fmt.Errorf("puredns bruteforce output path is required")
		}
		return outputPath, executor.bruteforceOutput, nil
	case command.Binary == "puredns" && len(command.Args) > 0 && command.Args[0] == "resolve":
		outputPath, ok := commandArgumentValue(command.Args, "--write")
		if !ok {
			return "", "", fmt.Errorf("puredns resolve output path is required")
		}
		return outputPath, executor.resolveOutput, nil
	default:
		return "", "", fmt.Errorf("unexpected tool invocation: %s", command.Binary)
	}
}

func invocationBinaries(invocations []toolInvocation) []string {
	binaries := make([]string, 0, len(invocations))
	for _, invocation := range invocations {
		binaries = append(binaries, invocation.Binary)
	}
	return binaries
}

func commandArgumentValue(args []string, flag string) (string, bool) {
	for index, arg := range args {
		if arg == flag && index+1 < len(args) {
			return args[index+1], true
		}
	}
	return "", false
}

func hasArgumentPrefix(args []string, prefix string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return true
		}
	}
	return false
}
