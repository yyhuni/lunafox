package subdomain_discovery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
)

type smokeRunner struct{}

func (runner *smokeRunner) Run(_ context.Context, cmd activity.Command) *activity.Result {
	return runner.execute(cmd)
}

func (runner *smokeRunner) RunParallel(_ context.Context, commands []activity.Command) []*activity.Result {
	results := make([]*activity.Result, 0, len(commands))
	for _, command := range commands {
		cmd := command
		results = append(results, runner.execute(cmd))
	}
	return results
}

func (runner *smokeRunner) RunSequential(_ context.Context, commands []activity.Command) []*activity.Result {
	results := make([]*activity.Result, 0, len(commands))
	for _, command := range commands {
		cmd := command
		results = append(results, runner.execute(cmd))
	}
	return results
}

func (runner *smokeRunner) execute(cmd activity.Command) *activity.Result {
	if strings.TrimSpace(cmd.OutputFile) != "" {
		_ = os.MkdirAll(filepath.Dir(cmd.OutputFile), 0755)
		_ = os.WriteFile(cmd.OutputFile, []byte("a.example.com\nb.example.com\n"), 0644)
	}

	return &activity.Result{
		Name:       cmd.Name,
		OutputFile: cmd.OutputFile,
		LogFile:    cmd.LogFile,
		ExitCode:   0,
	}
}

func TestWorkflowSmokeE2E_ConfigScheduleExecuteWriteback(t *testing.T) {
	withNopLogger(t)
	client := &capturePostClient{}
	workDir := t.TempDir()

	w := New(workDir)
	w.runner = &smokeRunner{}

	params := &workflow.Params{
		ScanID:       1,
		TargetID:     2,
		TargetType:   "domain",
		TargetName:   "example.com",
		WorkDir:      workDir,
		ScanConfig:   validScanConfig(),
		ServerClient: client,
	}

	output, err := w.Execute(params)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.NotNil(t, output.Metrics)
	require.Empty(t, output.Metrics.FailedTools)

	err = w.SaveResults(context.Background(), client, params, output)
	require.NoError(t, err)
	require.Greater(t, output.Metrics.ProcessedCount, 0)
	require.Greater(t, client.calls, 0)
}

func TestWorkflowSmoke_CodeFirstExecutionPathOnly(t *testing.T) {
	_, err := os.Stat(filepath.Join(".", "subdomain_command_builder.go"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	entries, err := filepath.Glob("*.go")
	require.NoError(t, err)
	for _, entry := range entries {
		if strings.HasSuffix(entry, "_test.go") {
			continue
		}
		data, readErr := os.ReadFile(entry)
		require.NoError(t, readErr)
		content := string(data)
		require.NotContains(t, content, "NewCommandBuilder(")
		require.NotContains(t, content, "MapConfigKeys(")
	}
}
