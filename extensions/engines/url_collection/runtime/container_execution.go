package urlcollectionruntime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// toolExecutor is deliberately narrow so pipeline tests can assert failure,
// timeout, and sibling-cancellation behavior without granting business code
// any host command or workspace capability.
type toolExecutor interface {
	run(context.Context, toolCommand) error
}

type containerToolExecutor struct{}

// run starts only an image-local named tool. The timeout context is derived
// from the task context, so the first of task cancellation/budget or tool
// timeout terminates the process without a stage-level retry.
func (containerToolExecutor) run(ctx context.Context, command toolCommand) error {
	if ctx == nil {
		return fmt.Errorf("tool context is required")
	}
	if command.name == "" || command.timeout <= 0 {
		return fmt.Errorf("tool name and positive timeout are required")
	}
	toolCtx, cancel := context.WithTimeout(ctx, command.timeout)
	defer cancel()
	process := exec.CommandContext(toolCtx, command.name, command.args...)
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	if err := process.Run(); err != nil {
		if timeoutErr := toolCtx.Err(); timeoutErr != nil {
			return fmt.Errorf("run %s: %w", command.name, timeoutErr)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("run %s: exit code %d", command.name, exitErr.ExitCode())
		}
		return fmt.Errorf("run %s: %w", command.name, err)
	}
	return nil
}
