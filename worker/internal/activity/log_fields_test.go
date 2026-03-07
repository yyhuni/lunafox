package activity

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func withObservedActivityLogger(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	core, logs := observer.New(zap.DebugLevel)
	prev := pkg.Logger
	pkg.Logger = zap.New(core)
	t.Cleanup(func() {
		pkg.Logger = prev
	})
	return logs
}

func TestRunnerStreamOutputLogsSemanticBufferMaxBytes(t *testing.T) {
	logs := withObservedActivityLogger(t)
	runner := NewRunner(t.TempDir())
	longLine := bytes.Repeat([]byte("a"), ScannerMaxBufSize+128)
	payload := append(longLine, '\n')

	_ = streamToLogFile(t, runner, bytes.NewReader(payload))

	entries := logs.FilterMessage("Activity output line truncated").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 truncation log, got %d", len(entries))
	}
	ctx := entries[0].ContextMap()
	if _, ok := ctx["buffer.max_bytes"]; !ok {
		t.Fatalf("expected buffer.max_bytes field, got %v", ctx)
	}
	if _, ok := ctx["maxBytes"]; ok {
		t.Fatalf("expected legacy maxBytes removed, got %v", ctx)
	}
}

func TestRunnerRunLogsSemanticProcessExitCode(t *testing.T) {
	logs := withObservedActivityLogger(t)
	r := NewRunner(t.TempDir())

	res := r.Run(context.Background(), Command{
		Name:    "test",
		Binary:  "cat",
		Args:    []string{"/definitely/not/exist"},
		Timeout: time.Second,
	})
	if res.Error == nil {
		t.Fatalf("expected run to fail")
	}

	entries := logs.FilterMessage("Activity failed").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 activity failure log, got %d", len(entries))
	}
	ctx := entries[0].ContextMap()
	if _, ok := ctx["process.exit_code"]; !ok {
		t.Fatalf("expected process.exit_code field, got %v", ctx)
	}
	if _, ok := ctx["exitCode"]; ok {
		t.Fatalf("expected legacy exitCode removed, got %v", ctx)
	}
}
