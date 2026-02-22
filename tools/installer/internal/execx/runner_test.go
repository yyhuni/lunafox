package execx

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestOSRunnerRunStreamsAndCapturesOutput(t *testing.T) {
	runner := NewOSRunner()
	var liveOut bytes.Buffer
	var liveErr bytes.Buffer

	command := Command{
		Name:         "sh",
		Args:         []string{"-c", "echo live-stdout; echo live-stderr 1>&2"},
		StdoutWriter: &liveOut,
		StderrWriter: &liveErr,
	}
	result, err := runner.Run(context.Background(), command)
	if err != nil {
		t.Fatalf("runner.Run failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "live-stdout") {
		t.Fatalf("expected buffered stdout, got=%q", result.Stdout)
	}
	if !strings.Contains(result.Stderr, "live-stderr") {
		t.Fatalf("expected buffered stderr, got=%q", result.Stderr)
	}
	if !strings.Contains(liveOut.String(), "live-stdout") {
		t.Fatalf("expected live stdout stream, got=%q", liveOut.String())
	}
	if !strings.Contains(liveErr.String(), "live-stderr") {
		t.Fatalf("expected live stderr stream, got=%q", liveErr.String())
	}
}
