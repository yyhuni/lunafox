package portscanruntime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// ProgressReporter is the narrow progress surface used by the Engine API v2
// container handler.
type ProgressReporter interface {
	Report(context.Context, string) error
}

type naabuExecutor interface {
	run(context.Context, naabuCommand) error
}

type containerNaabuExecutor struct{}

func (containerNaabuExecutor) run(ctx context.Context, command naabuCommand) error {
	if ctx == nil {
		return fmt.Errorf("naabu context is required")
	}
	if command.timeout <= 0 {
		return fmt.Errorf("naabu timeout must be positive")
	}

	commandContext, cancel := context.WithTimeout(ctx, command.timeout)
	defer cancel()
	process := exec.CommandContext(commandContext, "naabu", command.args...)
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	if err := process.Run(); err != nil {
		if contextErr := commandContext.Err(); contextErr != nil {
			return fmt.Errorf("run naabu %s: %w", command.label, contextErr)
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("run naabu %s: exit code %d", command.label, exitError.ExitCode())
		}
		return fmt.Errorf("run naabu %s: %w", command.label, err)
	}
	return nil
}
