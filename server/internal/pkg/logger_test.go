package pkg

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zapcore.Level
	}{
		{input: "debug", expected: zapcore.DebugLevel},
		{input: "info", expected: zapcore.InfoLevel},
		{input: "warn", expected: zapcore.WarnLevel},
		{input: "warning", expected: zapcore.WarnLevel},
		{input: "error", expected: zapcore.ErrorLevel},
		{input: "fatal", expected: zapcore.FatalLevel},
		{input: "unknown", expected: zapcore.InfoLevel},
	}

	for _, tt := range tests {
		if got := parseLogLevel(tt.input); got != tt.expected {
			t.Fatalf("expected %v for %q, got %v", tt.expected, tt.input, got)
		}
	}
}

func TestInitLogger(t *testing.T) {
	if err := InitLogger(&LogConfig{Level: "debug", Format: "json"}); err != nil {
		t.Fatalf("init logger failed: %v", err)
	}
	if Logger == nil || Sugar == nil {
		t.Fatalf("expected logger to be initialized")
	}
	Sync()
}

func TestInitDefaultLogger(t *testing.T) {
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("LOG_FORMAT", "json")

	if err := InitDefaultLogger(); err != nil {
		t.Fatalf("init default logger failed: %v", err)
	}
	if Logger == nil {
		t.Fatalf("expected logger to be initialized")
	}
	Sync()
}

func TestWithRequestIDUsesSemanticField(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)
	previousLogger := Logger
	previousSugar := Sugar
	Logger = logger
	Sugar = logger.Sugar()
	defer func() {
		Logger = previousLogger
		Sugar = previousSugar
	}()

	WithRequestID("req-123").Info("with request id")
	entries := logs.FilterMessage("with request id").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	ctx := entries[0].ContextMap()
	if _, ok := ctx["request.id"]; !ok {
		t.Fatalf("expected request.id field, got %v", ctx)
	}
	if _, ok := ctx["request_id"]; ok {
		t.Fatalf("expected legacy request_id field removed, got %v", ctx)
	}
}
