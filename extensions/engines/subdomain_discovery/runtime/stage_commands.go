package subdomaindiscoveryruntime

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	commandExitCodeTimeout = -1
	commandExitCodeError   = -2
)

type commandResult struct {
	Label              string
	OutputArtifactPath string
	ExitCode           int
	Error              error
	Duration           time.Duration
	OutputLineCount    int
}

type toolInvocation struct {
	Label          string
	Binary         string
	Args           []string
	TimeoutSeconds int64
}

type toolExecutionResult struct {
	ExitCode int
	Error    error
}

type toolExecutor interface {
	execute(context.Context, toolInvocation) (toolExecutionResult, error)
}

func buildToolInvocation(binary string, args []string, timeoutSeconds int64) toolInvocation {
	return toolInvocation{
		Label:          binary,
		Binary:         binary,
		Args:           append([]string(nil), args...),
		TimeoutSeconds: timeoutSeconds,
	}
}

func executeStageCommand(
	execCtx context.Context,
	run *discoveryRun,
	stageName string,
	command toolInvocation,
	outputArtifactPath string,
) commandResult {
	if err := execCtx.Err(); err != nil {
		return contextCancelledCommandResult(stageName, command, err)
	}
	if err := reportCommandStart(run, stageName, command); err != nil {
		return failedCommandResult(stageName, command, err)
	}
	startedAt := time.Now()
	executionResult, err := run.executor.execute(execCtx, command)
	if err != nil {
		result := failedCommandResult(stageName, command, err)
		result.Duration = time.Since(startedAt)
		reportCommandFinish(run, stageName, command, result)
		return result
	}
	result := commandResultFromExecution(stageName, command, outputArtifactPath, executionResult)
	result.Duration = time.Since(startedAt)
	if result.OutputArtifactPath != "" {
		result.OutputLineCount = countFileLines(result.OutputArtifactPath)
	}
	reportCommandFinish(run, stageName, command, result)
	return result
}

func reportCommandStart(run *discoveryRun, stageName string, command toolInvocation) error {
	if run == nil || run.progress == nil {
		return nil
	}
	return run.progress.report(fmt.Sprintf(
		"start command stage=%s label=%s command=%s",
		stageName,
		commandLabelForProgress(command),
		renderToolInvocation(command),
	))
}

func reportCommandFinish(run *discoveryRun, stageName string, command toolInvocation, result commandResult) {
	if run == nil || run.progress == nil {
		return
	}
	duration := result.Duration.Round(time.Millisecond).String()
	label := commandLabelForProgress(command)
	if result.Error != nil {
		_ = run.progress.report(fmt.Sprintf(
			"fail command stage=%s label=%s exitCode=%d duration=%s reason=%s",
			stageName,
			label,
			result.ExitCode,
			duration,
			result.Error.Error(),
		))
		return
	}
	_ = run.progress.report(fmt.Sprintf(
		"complete command stage=%s label=%s exitCode=%d duration=%s outputLines=%d",
		stageName,
		label,
		result.ExitCode,
		duration,
		result.OutputLineCount,
	))
}

func commandLabelForProgress(command toolInvocation) string {
	label := strings.TrimSpace(command.Label)
	if label != "" {
		return label
	}
	return strings.TrimSpace(command.Binary)
}

func renderToolInvocation(command toolInvocation) string {
	parts := []string{quoteCommandPartForProgress(command.Binary)}
	for _, arg := range command.Args {
		parts = append(parts, quoteCommandPartForProgress(arg))
	}
	return strings.Join(parts, " ")
}

func quoteCommandPartForProgress(value string) string {
	if value == "" {
		return "''"
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' || r == '/' || r == ':' || r == '=' || r == '@' || r == '%')
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func commandResultFromExecution(stageName string, command toolInvocation, outputArtifactPath string, executionResult toolExecutionResult) commandResult {
	result := commandResult{
		Label:              commandResultLabel(stageName, command),
		OutputArtifactPath: outputArtifactPath,
		ExitCode:           executionResult.ExitCode,
	}
	if executionResult.Error != nil {
		result.Error = executionResult.Error
	}
	return result
}

func failedCommandResult(stageName string, command toolInvocation, err error) commandResult {
	return commandResult{
		Label:    commandResultLabel(stageName, command),
		ExitCode: commandExitCodeError,
		Error:    err,
	}
}

func contextCancelledCommandResult(stageName string, command toolInvocation, err error) commandResult {
	return commandResult{
		Label:    commandResultLabel(stageName, command),
		ExitCode: commandExitCodeError,
		Error:    fmt.Errorf("context cancelled: %w", err),
	}
}

func commandResultLabel(stageName string, command toolInvocation) string {
	label := strings.TrimSpace(command.Label)
	if label == "" {
		return stageName
	}
	return fmt.Sprintf("%s/%s", stageName, label)
}

// collectCommandOutcomes folds image-owned tool outcomes into a stage outcome.
func collectCommandOutcomes(results []commandResult) stageOutcome {
	var outcome stageOutcome
	for _, r := range results {
		if r.Error != nil {
			outcome.failed = append(outcome.failed, r.Label)
			if outcome.failureCause == nil {
				outcome.failureCause = r.Error
			}
			continue
		}
		outcome.success = append(outcome.success, r.Label)
		if r.OutputArtifactPath != "" {
			outcome.outputArtifactPaths = append(outcome.outputArtifactPaths, r.OutputArtifactPath)
		}
	}
	return outcome
}
