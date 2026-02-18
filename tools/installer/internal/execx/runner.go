package execx

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Command struct {
	Name  string
	Args  []string
	Env   []string
	Dir   string
	Stdin string
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
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

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
