package urlcollectionruntime

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestContainerToolExecutorDistinguishesTaskCancellationFromToolTimeout(t *testing.T) {
	command := toolCommand{name: "sh", args: []string{"-c", "sleep 1"}, timeout: time.Second}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (containerToolExecutor{}).run(cancelled, command); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled tool error = %v, want context cancellation", err)
	}

	command.timeout = 5 * time.Millisecond
	if err := (containerToolExecutor{}).run(context.Background(), command); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timed-out tool error = %v, want deadline exceeded", err)
	}
}

func TestContainerToolExecutorPreservesExistingRawToolDiagnostics(t *testing.T) {
	previousStdout, previousStderr := os.Stdout, os.Stderr
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = stdoutWriter, stderrWriter
	t.Cleanup(func() {
		os.Stdout, os.Stderr = previousStdout, previousStderr
	})

	err = (containerToolExecutor{}).run(context.Background(), toolCommand{
		name: "sh", args: []string{"-c", "printf stdout-diagnostic; printf stderr-diagnostic >&2; exit 7"}, timeout: time.Second,
	})
	if err == nil || !strings.Contains(err.Error(), "exit code 7") {
		t.Fatalf("tool failure = %v", err)
	}
	if err := stdoutWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := stderrWriter.Close(); err != nil {
		t.Fatal(err)
	}
	stdout, readErr := io.ReadAll(stdoutReader)
	if readErr != nil {
		t.Fatal(readErr)
	}
	stderr, readErr := io.ReadAll(stderrReader)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(stdout) != "stdout-diagnostic" || string(stderr) != "stderr-diagnostic" {
		t.Fatalf("raw diagnostics stdout/stderr = %q/%q", stdout, stderr)
	}
}
