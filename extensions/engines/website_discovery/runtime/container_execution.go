package websitediscoveryruntime

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

type httpxExecutor interface {
	run(context.Context, httpxCommand) error
}

type containerHTTPXExecutor struct{}

func (containerHTTPXExecutor) run(ctx context.Context, command httpxCommand) error {
	if ctx == nil {
		return fmt.Errorf("httpx context is required")
	}
	if command.timeout <= 0 {
		return fmt.Errorf("httpx timeout must be positive")
	}

	commandContext, cancel := context.WithTimeout(ctx, command.timeout)
	defer cancel()
	process := exec.CommandContext(commandContext, "httpx", command.args...)
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	if err := process.Run(); err != nil {
		if contextErr := commandContext.Err(); contextErr != nil {
			return fmt.Errorf("run httpx: %w", contextErr)
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("run httpx: exit code %d", exitError.ExitCode())
		}
		return fmt.Errorf("run httpx: %w", err)
	}
	return nil
}
