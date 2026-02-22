package execx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/creack/pty"
)

type Command struct {
	Name  string
	Args  []string
	Env   []string
	Dir   string
	Stdin string

	// PreferTTY enables PTY execution when current process has an interactive terminal.
	PreferTTY bool

	// Optional live output writers. When nil, defaults to os.Stdout/os.Stderr.
	// Output is still buffered into Result.
	StdoutWriter io.Writer
	StderrWriter io.Writer
}

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Runner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, command Command) (Result, error)
}

type OSRunner struct{}

func NewOSRunner() *OSRunner {
	return &OSRunner{}
}

func (runner *OSRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (runner *OSRunner) Run(ctx context.Context, command Command) (Result, error) {
	if shouldUsePTY(command) {
		return runner.runPTY(ctx, command)
	}
	return runner.runPipe(ctx, command)
}

func (runner *OSRunner) runPipe(ctx context.Context, command Command) (Result, error) {
	execCmd := exec.CommandContext(ctx, command.Name, command.Args...)
	if len(command.Env) > 0 {
		execCmd.Env = command.Env
	}
	if command.Dir != "" {
		execCmd.Dir = command.Dir
	}
	if command.Stdin != "" {
		execCmd.Stdin = strings.NewReader(command.Stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	stdoutLive := command.StdoutWriter
	if stdoutLive == nil {
		stdoutLive = os.Stdout
	}
	stdoutWriters := []io.Writer{&stdout}
	if stdoutLive != nil {
		stdoutWriters = append(stdoutWriters, stdoutLive)
	}
	execCmd.Stdout = io.MultiWriter(stdoutWriters...)

	stderrLive := command.StderrWriter
	if stderrLive == nil {
		stderrLive = os.Stderr
	}
	stderrWriters := []io.Writer{&stderr}
	if stderrLive != nil {
		stderrWriters = append(stderrWriters, stderrLive)
	}
	execCmd.Stderr = io.MultiWriter(stderrWriters...)

	err := execCmd.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: 0}
	if err == nil {
		return result, nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	} else {
		result.ExitCode = -1
	}

	return result, &ExecError{Command: command, Result: result, Cause: err}
}

func (runner *OSRunner) runPTY(ctx context.Context, command Command) (Result, error) {
	execCmd := exec.CommandContext(ctx, command.Name, command.Args...)
	if len(command.Env) > 0 {
		execCmd.Env = command.Env
	}
	if command.Dir != "" {
		execCmd.Dir = command.Dir
	}

	ptmx, err := pty.Start(execCmd)
	if err != nil {
		result := Result{ExitCode: -1}
		return result, &ExecError{Command: command, Result: result, Cause: err}
	}
	defer func() {
		_ = ptmx.Close()
	}()

	if isTerminal(os.Stdin) {
		_ = pty.InheritSize(os.Stdin, ptmx)
	}

	liveWriter := command.StdoutWriter
	if liveWriter == nil {
		liveWriter = command.StderrWriter
	}
	if liveWriter == nil {
		liveWriter = os.Stdout
	}

	var stdout bytes.Buffer
	copyErrCh := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(io.MultiWriter(&stdout, liveWriter), ptmx)
		copyErrCh <- copyErr
	}()

	waitErr := execCmd.Wait()
	_ = ptmx.Close()
	copyErr := <-copyErrCh
	if isIgnorablePTYReadError(copyErr) {
		copyErr = nil
	}

	result := Result{
		Stdout:   stdout.String(),
		Stderr:   "",
		ExitCode: 0,
	}

	if waitErr == nil {
		if copyErr != nil {
			result.ExitCode = -1
			return result, &ExecError{Command: command, Result: result, Cause: copyErr}
		}
		return result, nil
	}

	if exitErr, ok := waitErr.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	} else {
		result.ExitCode = -1
	}
	return result, &ExecError{Command: command, Result: result, Cause: waitErr}
}

func shouldUsePTY(command Command) bool {
	if !command.PreferTTY {
		return false
	}
	if command.Stdin != "" {
		return false
	}
	if runtime.GOOS == "windows" {
		return false
	}
	return strings.TrimSpace(os.Getenv("LUNAFOX_DISABLE_PTY")) != "1"
}

func isTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func isIgnorablePTYReadError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, io.EOF) || errors.Is(err, syscall.EIO)
}

type ExecError struct {
	Command Command
	Result  Result
	Cause   error
}

func (err *ExecError) Error() string {
	return fmt.Sprintf("命令执行失败: %s %s", err.Command.Name, strings.Join(err.Command.Args, " "))
}

func (err *ExecError) Unwrap() error {
	return err.Cause
}
