package activity

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"go.uber.org/zap"
)

type commandExecution struct {
	execCmd *exec.Cmd
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	logFile *os.File
}

// Run executes a single activity with streaming output
func (r *Runner) Run(ctx context.Context, cmd Command) *Result {
	start := time.Now()
	result := newRunResult(cmd)

	if err := validateRunInputs(ctx, cmd); err != nil {
		setRunError(result, ExitCodeError, err)
		return result
	}

	// Limit external command process concurrency per worker container.
	if err := r.acquire(ctx); err != nil {
		setRunError(result, ExitCodeError, fmt.Errorf("context cancelled while waiting for command slot: %w", err))
		return result
	}
	defer r.release()

	execCtx, cancel := context.WithTimeout(ctx, cmd.Timeout)
	defer cancel()

	execution, err := r.prepareExecution(execCtx, cmd)
	if err != nil {
		setRunError(result, ExitCodeError, err)
		return result
	}

	if execution.logFile != nil {
		defer func() { _ = execution.logFile.Close() }()
		r.writeLogHeader(execution.logFile, cmd)
	}

	if err := execution.execCmd.Start(); err != nil {
		setRunError(result, ExitCodeError, fmt.Errorf("failed to start command: %w", err))
		return result
	}

	// Ensure process cleanup on any exit path.
	defer killProcessGroup(execution.execCmd)

	r.streamCommandOutput(execution, cmd)
	waitErr := execution.execCmd.Wait()
	result.Duration = time.Since(start)

	if execution.logFile != nil {
		r.writeLogFooter(execution.logFile, result)
	}

	r.finalizeRunResult(execCtx, cmd, waitErr, result)
	return result
}

func newRunResult(cmd Command) *Result {
	return &Result{
		Name:       cmd.Name,
		OutputFile: cmd.OutputFile,
		LogFile:    cmd.LogFile,
	}
}

func setRunError(result *Result, exitCode int, err error) {
	result.ExitCode = exitCode
	result.Error = err
}

func validateRunInputs(ctx context.Context, cmd Command) error {
	if ctx.Err() != nil {
		return fmt.Errorf("context cancelled before execution: %w", ctx.Err())
	}
	if cmd.Timeout <= 0 {
		return fmt.Errorf("invalid timeout %v: must be > 0", cmd.Timeout)
	}
	return nil
}

func (r *Runner) prepareExecution(execCtx context.Context, cmd Command) (*commandExecution, error) {
	execCmd := exec.CommandContext(execCtx, "sh", "-c", cmd.Command)
	execCmd.Dir = r.workDir
	// Create new process group so we can kill all child processes.
	execCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := execCmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	return &commandExecution{
		execCmd: execCmd,
		stdout:  stdout,
		stderr:  stderr,
		logFile: r.prepareLogFile(cmd),
	}, nil
}

func (r *Runner) streamCommandOutput(execution *commandExecution, cmd Command) {
	var wg sync.WaitGroup
	wg.Add(2)

	go r.streamOutput(&wg, execution.stdout, execution.logFile, cmd.Name, "stdout")
	go r.streamOutput(&wg, execution.stderr, execution.logFile, cmd.Name, "stderr")

	wg.Wait()
}

func (r *Runner) finalizeRunResult(execCtx context.Context, cmd Command, waitErr error, result *Result) {
	if execCtx.Err() == context.DeadlineExceeded {
		setRunError(result, ExitCodeTimeout, fmt.Errorf("activity execution timeout after %v", cmd.Timeout))
		pkg.Logger.Error("Activity timeout",
			zap.String("activity", cmd.Name),
			zap.Duration("timeout", cmd.Timeout))
		return
	}

	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = ExitCodeError
		}
		result.Error = fmt.Errorf("activity execution failed: %w", waitErr)
		pkg.Logger.Error("Activity failed",
			zap.String("activity", cmd.Name),
			zap.Int("exitCode", result.ExitCode),
			zap.Error(waitErr))
		return
	}

	result.ExitCode = 0
	pkg.Logger.Info("Activity completed",
		zap.String("activity", cmd.Name),
		zap.Duration("duration", result.Duration))
}

func (r *Runner) prepareLogFile(cmd Command) *os.File {
	if cmd.LogFile == "" {
		return nil
	}

	dir := filepath.Dir(cmd.LogFile)
	if err := os.MkdirAll(dir, DefaultDirPerm); err != nil {
		pkg.Logger.Warn("Failed to create log directory",
			zap.String("activity", cmd.Name),
			zap.Error(err))
		return nil
	}

	f, err := os.Create(cmd.LogFile)
	if err != nil {
		pkg.Logger.Warn("Failed to create log file",
			zap.String("activity", cmd.Name),
			zap.Error(err))
		return nil
	}

	return f
}

const logSeparator = "============================================================"

func (r *Runner) writeLogHeader(f *os.File, cmd Command) {
	_, _ = fmt.Fprintf(f, "$ %s\n", cmd.Command)
	_, _ = fmt.Fprintln(f, logSeparator)
	_, _ = fmt.Fprintf(f, "# Tool: %s\n", cmd.Name)
	_, _ = fmt.Fprintf(f, "# Started: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	_, _ = fmt.Fprintf(f, "# Timeout: %v\n", cmd.Timeout)
	_, _ = fmt.Fprintln(f, "# Status: Running...")
	_, _ = fmt.Fprintln(f, logSeparator)
	_, _ = fmt.Fprintln(f)
}

func (r *Runner) writeLogFooter(f *os.File, result *Result) {
	status := "✓ Success"
	if result.ExitCode != 0 {
		status = "✗ Failed"
	}

	_, _ = fmt.Fprintln(f)
	_, _ = fmt.Fprintln(f, logSeparator)
	_, _ = fmt.Fprintf(f, "# Finished: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	_, _ = fmt.Fprintf(f, "# Duration: %.2fs\n", result.Duration.Seconds())
	_, _ = fmt.Fprintf(f, "# Exit Code: %d\n", result.ExitCode)
	_, _ = fmt.Fprintf(f, "# Status: %s\n", status)
	_, _ = fmt.Fprintln(f, logSeparator)
}
