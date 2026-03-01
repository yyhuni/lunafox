package subdomain_discovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
)

func TestRunStageCommandsSequential(t *testing.T) {
	withNopLogger(t)
	t.Setenv(activity.EnvMaxCmdConcurrency, "2")

	workDir := t.TempDir()

	w := &Workflow{
		runner: activity.NewRunner(workDir),
		stageMetadata: map[string]workflow.StageMetadata{
			"seq": {ID: "seq", Parallel: false},
		},
	}

	ctx := &workflowContext{ctx: context.Background()}
	cmds := []activity.Command{
		{Name: "c1", Binary: "sleep", Args: []string{"0.2"}, Timeout: time.Second},
		{Name: "c2", Binary: "sleep", Args: []string{"0.2"}, Timeout: time.Second},
	}

	start := time.Now()
	results := w.runStageCommands(ctx, "seq", cmds)
	elapsed := time.Since(start)
	require.Len(t, results, 2)
	for _, res := range results {
		require.NoError(t, res.Error)
		require.Equal(t, 0, res.ExitCode)
	}
	require.GreaterOrEqual(t, elapsed, 350*time.Millisecond)
}

func TestRunStageCommandsParallel(t *testing.T) {
	withNopLogger(t)
	t.Setenv(activity.EnvMaxCmdConcurrency, "2")

	workDir := t.TempDir()

	w := &Workflow{
		runner: activity.NewRunner(workDir),
		stageMetadata: map[string]workflow.StageMetadata{
			"par": {ID: "par", Parallel: true},
		},
	}

	ctx := &workflowContext{ctx: context.Background()}
	cmds := []activity.Command{
		{Name: "c1", Binary: "sleep", Args: []string{"0.3"}, Timeout: time.Second},
		{Name: "c2", Binary: "sleep", Args: []string{"0.3"}, Timeout: time.Second},
	}

	start := time.Now()
	results := w.runStageCommands(ctx, "par", cmds)
	elapsed := time.Since(start)
	require.Len(t, results, 2)
	for _, res := range results {
		require.NoError(t, res.Error)
		require.Equal(t, 0, res.ExitCode)
	}
	require.Less(t, elapsed, 550*time.Millisecond)
}
