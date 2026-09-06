package fingerprintdetectionruntime

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type observerWardExecutor interface {
	Run(context.Context, observerWardCommand, func([]byte) error) error
}

type containerObserverWardExecutor struct{}

func (containerObserverWardExecutor) Run(ctx context.Context, command observerWardCommand, visit func([]byte) error) error {
	if ctx == nil {
		return errors.New("Observer Ward context is required")
	}
	if command.Name == "" || command.Timeout <= 0 || visit == nil {
		return errors.New("Observer Ward command is incomplete")
	}
	commandContext, cancel := context.WithTimeout(ctx, command.Timeout)
	defer cancel()
	process := exec.CommandContext(commandContext, command.Name, command.Args...)
	stdout, err := process.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open Observer Ward stdout: %w", err)
	}
	process.Stderr = os.Stderr
	if err := process.Start(); err != nil {
		return fmt.Errorf("start Observer Ward: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	visitErr := error(nil)
	for scanner.Scan() {
		if err := visit(append([]byte(nil), scanner.Bytes()...)); err != nil {
			visitErr = err
			break
		}
	}
	if scanErr := scanner.Err(); scanErr != nil && visitErr == nil {
		visitErr = scanErr
	}
	if visitErr != nil {
		_ = process.Process.Kill()
		_ = process.Wait()
		return visitErr
	}
	if err := process.Wait(); err != nil {
		if timeoutErr := commandContext.Err(); timeoutErr != nil {
			return fmt.Errorf("Observer Ward execution: %w", timeoutErr)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("Observer Ward exited with code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("Observer Ward execution: %w", err)
	}
	if err := commandContext.Err(); err != nil {
		return err
	}
	return nil
}
