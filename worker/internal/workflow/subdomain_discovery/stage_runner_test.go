package subdomain_discovery

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
	"github.com/stretchr/testify/require"
)

func TestRunStageCommandsSequential(t *testing.T) {
	withNopLogger(t)
	t.Setenv(activity.EnvMaxCmdConcurrency, "2")

	workDir := t.TempDir()
	lockDir := filepath.Join(workDir, "lock")
	cmdStr := fmt.Sprintf("mkdir %q || exit 99; sleep 0.2; rmdir %q", lockDir, lockDir)

	w := &Workflow{
		runner: activity.NewRunner(workDir),
		stageMetadata: map[string]workflow.StageMetadata{
			"seq": {ID: "seq", Parallel: false},
		},
	}

	ctx := &workflowContext{ctx: context.Background()}
	cmds := []activity.Command{
		{Name: "c1", Command: cmdStr, Timeout: time.Second},
		{Name: "c2", Command: cmdStr, Timeout: time.Second},
	}

	results := w.runStageCommands(ctx, "seq", cmds)
	require.Len(t, results, 2)
	for _, res := range results {
		require.NoError(t, res.Error)
		require.Equal(t, 0, res.ExitCode)
	}
}

func TestRunStageCommandsParallel(t *testing.T) {
	withNopLogger(t)
	t.Setenv(activity.EnvMaxCmdConcurrency, "2")

	workDir := t.TempDir()
	lockDir := filepath.Join(workDir, "lock")
	cmdStr := fmt.Sprintf("mkdir %q || exit 99; sleep 0.3; rmdir %q", lockDir, lockDir)

	w := &Workflow{
		runner: activity.NewRunner(workDir),
		stageMetadata: map[string]workflow.StageMetadata{
			"par": {ID: "par", Parallel: true},
		},
	}

	ctx := &workflowContext{ctx: context.Background()}
	cmds := []activity.Command{
		{Name: "c1", Command: cmdStr, Timeout: time.Second},
		{Name: "c2", Command: cmdStr, Timeout: time.Second},
	}

	results := w.runStageCommands(ctx, "par", cmds)
	require.Len(t, results, 2)
	failed := 0
	for _, res := range results {
		if res.Error != nil {
			failed++
		}
	}
	require.GreaterOrEqual(t, failed, 1)
}
