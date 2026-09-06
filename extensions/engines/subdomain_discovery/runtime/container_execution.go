package subdomaindiscoveryruntime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ProgressReporter is the narrow progress surface used by the Engine API v2
// container handler.
type ProgressReporter interface {
	Report(context.Context, string) error
}

type executionProgress interface {
	report(string) error
}

type contextProgress struct {
	executionContext context.Context
	progress         ProgressReporter
}

func newExecutionProgress(ctx context.Context, progress ProgressReporter) (executionProgress, error) {
	if ctx == nil {
		return nil, fmt.Errorf("execution context is required")
	}
	if progress == nil {
		return nil, fmt.Errorf("engine progress port is required")
	}
	return &contextProgress{executionContext: ctx, progress: progress}, nil
}

func (progress *contextProgress) report(message string) error {
	return progress.progress.Report(progress.executionContext, message)
}

type containerToolExecutor struct{}

func (containerToolExecutor) execute(ctx context.Context, command toolInvocation) (toolExecutionResult, error) {
	if ctx == nil {
		return toolExecutionResult{}, fmt.Errorf("process context is required")
	}
	if strings.TrimSpace(command.Binary) == "" {
		return toolExecutionResult{}, fmt.Errorf("process binary is required")
	}
	if command.TimeoutSeconds <= 0 {
		return toolExecutionResult{}, fmt.Errorf("process timeout must be positive")
	}

	commandContext, cancel := context.WithTimeout(ctx, time.Duration(command.TimeoutSeconds)*time.Second)
	defer cancel()
	process := exec.CommandContext(commandContext, command.Binary, command.Args...)
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	if err := process.Run(); err != nil {
		if contextErr := commandContext.Err(); contextErr != nil {
			// The generated facade recognizes signal cancellation through
			// errors.Is, so the tool boundary must retain the context cause.
			return toolExecutionResult{ExitCode: commandExitCodeTimeout, Error: contextErr}, fmt.Errorf("execute %s: %w", commandLabelForProgress(command), contextErr)
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			return toolExecutionResult{ExitCode: exitError.ExitCode(), Error: fmt.Errorf("%s exited with code %d: %w", commandLabelForProgress(command), exitError.ExitCode(), err)}, nil
		}
		return toolExecutionResult{ExitCode: commandExitCodeError, Error: err}, nil
	}
	return toolExecutionResult{ExitCode: 0}, nil
}
