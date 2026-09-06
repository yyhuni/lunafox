package audit

import (
	"reflect"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestEventAllowlistContainsOnlyNonPayloadMetadata(t *testing.T) {
	typ := reflect.TypeOf(Event{})
	for _, forbidden := range []string{"Bearer", "Secret", "Digest", "Hash", "Input", "Filter", "RawOutput", "Result", "Body", "SQL", "Stack", "Path"} {
		for fieldIndex := 0; fieldIndex < typ.NumField(); fieldIndex++ {
			if field := typ.Field(fieldIndex); field.Name == forbidden {
				t.Fatalf("audit event has forbidden field %q", forbidden)
			}
		}
	}
	for _, allowed := range []string{"RequestID", "UserID", "KeyRecordID", "Method", "Tool", "ResultCategory", "ErrorCode", "Duration", "ResponseBytes"} {
		if _, ok := typ.FieldByName(allowed); !ok {
			t.Fatalf("audit event lost allowlisted field %q", allowed)
		}
	}
}

func TestLoggerEmitsBoundedStructuredFields(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	previousLogger, previousSugar := pkg.Logger, pkg.Sugar
	pkg.Logger = zap.New(core)
	pkg.Sugar = pkg.Logger.Sugar()
	t.Cleanup(func() {
		pkg.Logger = previousLogger
		pkg.Sugar = previousSugar
	})

	NewLogger().Emit(Event{
		RequestID:      "req-1",
		UserID:         7,
		KeyRecordID:    9,
		Method:         "tools/call",
		Tool:           "list_targets",
		ResultCategory: "SUCCESS",
		ErrorCode:      "",
		Duration:       25 * time.Millisecond,
		ResponseBytes:  128,
	})

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("audit log count = %d", len(entries))
	}
	fields := entries[0].ContextMap()
	for _, key := range []string{"request.id", "mcp.method", "mcp.tool", "mcp.result_category", "mcp.duration_ms", "mcp.response_bytes", "mcp.user_id", "mcp.key_record_id"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("audit field %q missing: %#v", key, fields)
		}
	}
	for _, key := range []string{"authorization", "bearer", "mcp.key_digest", "mcp.input", "mcp.result", "response_body", "stack"} {
		if _, ok := fields[key]; ok {
			t.Fatalf("forbidden audit field %q present: %#v", key, fields)
		}
	}
}
