package screenshotruntime

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

// The container execution adapter is the only Screenshot Engine boundary that
// starts image-local tools. Runtime business code receives parsed rows and
// encoded bytes through these narrow interfaces, matching the other first-party
// Engines' process execution contract.
type httpxPassExecutor interface {
	Run(context.Context, httpxCommand, func([]byte) error) error
}

type containerHTTPXExecutor struct{}

func (containerHTTPXExecutor) Run(ctx context.Context, command httpxCommand, onRecord func([]byte) error) error {
	if ctx == nil || onRecord == nil {
		return errors.New("HTTPX process context and row callback are required")
	}
	// Production passes rely on the Agent task context for total lifecycle
	// cancellation. A command-local timeout remains available to process tests
	// and future callers, but page-timeout must never cap a whole candidate pass.
	commandContext := ctx
	cancel := func() {}
	if command.timeout > 0 {
		commandContext, cancel = context.WithTimeout(ctx, command.timeout)
	}
	defer cancel()
	processContext, cancelProcess := context.WithCancel(commandContext)
	defer cancelProcess()
	process := exec.CommandContext(processContext, "httpx", command.args...)
	stdout, err := process.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open HTTPX stdout: %w", err)
	}
	stderrBuffer := &limitedBuffer{limit: maxHTTPXStderrBytes, onLimit: cancelProcess}
	// Let os/exec own stderr copying so Wait can synchronize its completion.
	// Reading StderrPipe in a separate goroutine races with Wait closing that
	// pipe after a clean process exit and can surface a false "file already
	// closed" error.
	process.Stderr = stderrBuffer
	if err := process.Start(); err != nil {
		return fmt.Errorf("start HTTPX: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), maxHTTPXRecordBytes)
	var callbackErr error
	for scanner.Scan() {
		if err := onRecord(append([]byte(nil), scanner.Bytes()...)); err != nil {
			callbackErr = err
			break
		}
	}
	if scanErr := scanner.Err(); scanErr != nil && callbackErr == nil {
		callbackErr = fmt.Errorf("read HTTPX JSONL: %w", scanErr)
	}
	if callbackErr != nil {
		_ = process.Process.Kill()
	}
	waitErr := process.Wait()
	if callbackErr != nil {
		return callbackErr
	}
	if stderrErr := stderrBuffer.Err(); stderrErr != nil {
		return fmt.Errorf("read HTTPX stderr: %w", stderrErr)
	}
	if waitErr != nil {
		if contextErr := processContext.Err(); contextErr != nil {
			return fmt.Errorf("run HTTPX: %w", contextErr)
		}
		return fmt.Errorf("HTTPX exited unsuccessfully: %w", waitErr)
	}
	return nil
}

type cwebpRunner interface {
	Run(context.Context, []string) error
}

type containerCWebPRunner struct{}

func (containerCWebPRunner) Run(ctx context.Context, args []string) error {
	if ctx == nil {
		return errors.New("cwebp context is required")
	}
	command := exec.CommandContext(ctx, "cwebp", args...)
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		return fmt.Errorf("cwebp failed: %w", err)
	}
	return nil
}

type limitedBuffer struct {
	limit    int
	total    int
	exceeded bool
	onLimit  func()
}

func (buffer *limitedBuffer) Write(data []byte) (int, error) {
	if buffer.exceeded {
		return 0, errors.New("HTTPX stderr exceeds limit")
	}
	remaining := buffer.limit - buffer.total
	if remaining <= 0 {
		buffer.exceeded = true
		if buffer.onLimit != nil {
			buffer.onLimit()
		}
		return 0, errors.New("HTTPX stderr exceeds limit")
	}
	if len(data) > remaining {
		buffer.total += remaining
		buffer.exceeded = true
		if buffer.onLimit != nil {
			buffer.onLimit()
		}
		return remaining, errors.New("HTTPX stderr exceeds limit")
	}
	buffer.total += len(data)
	return len(data), nil
}

func (buffer *limitedBuffer) Err() error {
	if buffer.exceeded {
		return errors.New("HTTPX stderr exceeds limit")
	}
	return nil
}

// Keep the old test seam name as a type alias without creating a second
// process-execution implementation outside this adapter file.
type processHTTPX = containerHTTPXExecutor

// Keep the old cwebp test seam name as a type alias for the adapter.
type processCWebP = containerCWebPRunner
