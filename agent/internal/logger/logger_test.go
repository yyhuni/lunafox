package logger

import (
	"bytes"
	"strings"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestBuildLoggerRoutesLevelsToStreams(t *testing.T) {
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	logger, err := buildLogger("info", "release", zapcore.AddSync(&stdoutBuf), zapcore.AddSync(&stderrBuf))
	if err != nil {
		t.Fatalf("buildLogger returned error: %v", err)
	}

	logger.Info("info-msg")
	logger.Warn("warn-msg")
	_ = logger.Sync()

	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()

	if !strings.Contains(stdout, "\"msg\":\"info-msg\"") {
		t.Fatalf("expected info log in stdout, got: %s", stdout)
	}
	if strings.Contains(stdout, "\"msg\":\"warn-msg\"") {
		t.Fatalf("warn log should not be in stdout, got: %s", stdout)
	}
	if strings.Contains(stderr, "\"msg\":\"info-msg\"") {
		t.Fatalf("info log should not be in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "\"msg\":\"warn-msg\"") {
		t.Fatalf("expected warn log in stderr, got: %s", stderr)
	}
}

func TestBuildLoggerRespectsMinimumLevel(t *testing.T) {
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	logger, err := buildLogger("warn", "release", zapcore.AddSync(&stdoutBuf), zapcore.AddSync(&stderrBuf))
	if err != nil {
		t.Fatalf("buildLogger returned error: %v", err)
	}

	logger.Debug("debug-msg")
	logger.Info("info-msg")
	logger.Warn("warn-msg")
	_ = logger.Sync()

	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()

	if strings.Contains(stdout, "debug-msg") || strings.Contains(stdout, "info-msg") {
		t.Fatalf("logs below warn should be filtered from stdout, got: %s", stdout)
	}
	if strings.Contains(stderr, "debug-msg") || strings.Contains(stderr, "info-msg") {
		t.Fatalf("logs below warn should be filtered from stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "\"msg\":\"warn-msg\"") {
		t.Fatalf("expected warn log in stderr, got: %s", stderr)
	}
}
