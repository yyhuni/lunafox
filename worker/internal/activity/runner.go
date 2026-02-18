package activity

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"go.uber.org/zap"
)

const (
	DefaultDirPerm  = 0755
	ExitCodeTimeout = -1
	ExitCodeError   = -2

	// Runner concurrency control (external command processes)
	// Enforced per worker container/workflow instance.
	EnvMaxCmdConcurrency     = "WORKER_MAX_CMD_CONCURRENCY"
	DefaultMaxCmdConcurrency = 2

	// Scanner buffer sizes
	ScannerInitBufSize = 64 * 1024   // 64KB initial buffer
	ScannerMaxBufSize  = 1024 * 1024 // 1MB max buffer for long lines
)

// Result represents the result of an activity execution
type Result struct {
	Name       string
	OutputFile string
	LogFile    string
	ExitCode   int
	Duration   time.Duration
	Error      error
}

// Command represents a command to execute
type Command struct {
	Name       string
	Command    string
	OutputFile string
	LogFile    string
	Timeout    time.Duration
}

// Runner executes activities (external tools)
type Runner struct {
	workDir string
	sem     chan struct{}
}

// NewRunner creates a new activity runner
func NewRunner(workDir string) *Runner {
	maxConc := getMaxCmdConcurrency()
	return &Runner{
		workDir: workDir,
		sem:     make(chan struct{}, maxConc),
	}
}

func getMaxCmdConcurrency() int {
	v := os.Getenv(EnvMaxCmdConcurrency)
	if v == "" {
		return DefaultMaxCmdConcurrency
	}

	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		if pkg.Logger != nil {
			pkg.Logger.Warn("Invalid max command concurrency; using default",
				zap.String("env", EnvMaxCmdConcurrency),
				zap.String("value", v),
				zap.Int("default", DefaultMaxCmdConcurrency))
		}
		return DefaultMaxCmdConcurrency
	}

	return n
}

func (r *Runner) acquire(ctx context.Context) error {
	select {
	case r.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Runner) release() {
	<-r.sem
}

// killProcessGroup terminates the entire process group
// When using shell=true (sh -c), the actual tool runs as a child of the shell.
// If we only kill the shell process, the child becomes an orphan and keeps running.
// By killing the process group, we ensure all child processes are terminated.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	pid := cmd.Process.Pid

	// Try to kill the process group first
	// The negative PID signals the entire process group
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		pkg.Logger.Debug("Failed to kill process group, trying single process",
			zap.Int("pid", pid),
			zap.Error(err))
		// Fallback: kill single process
		_ = cmd.Process.Kill()
	} else {
		pkg.Logger.Debug("Killed process group", zap.Int("pgid", pid))
	}
}

// RunParallel executes multiple activities in parallel
func (r *Runner) RunParallel(ctx context.Context, commands []Command) []*Result {
	if len(commands) == 0 {
		return nil
	}

	results := make([]*Result, len(commands))
	var wg sync.WaitGroup

	for i, cmd := range commands {
		if ctx.Err() != nil {
			results[i] = &Result{
				Name:     cmd.Name,
				ExitCode: ExitCodeError,
				Error:    fmt.Errorf("context cancelled: %w", ctx.Err()),
			}
			continue
		}

		wg.Add(1)
		go func(idx int, c Command) {
			defer wg.Done()
			results[idx] = r.Run(ctx, c)
		}(i, cmd)
	}

	wg.Wait()
	return results
}

// RunSequential executes multiple activities sequentially (one after another)
func (r *Runner) RunSequential(ctx context.Context, commands []Command) []*Result {
	if len(commands) == 0 {
		return nil
	}

	results := make([]*Result, len(commands))

	for i, cmd := range commands {
		if ctx.Err() != nil {
			results[i] = &Result{
				Name:     cmd.Name,
				ExitCode: ExitCodeError,
				Error:    fmt.Errorf("context cancelled: %w", ctx.Err()),
			}
			continue
		}

		results[i] = r.Run(ctx, cmd)
	}

	return results
}
