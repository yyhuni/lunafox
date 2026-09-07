package database

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestNewDatabaseLoggerUsesSharedZapAdapterDefaults(t *testing.T) {
	databaseLogger, ok := newDatabaseLogger().(*gormZapLogger)
	if !ok {
		t.Fatalf("database logger type = %T, want *gormZapLogger", newDatabaseLogger())
	}
	if databaseLogger.config.LogLevel != logger.Warn {
		t.Fatalf("database logger level = %v, want Warn", databaseLogger.config.LogLevel)
	}
	if databaseLogger.config.SlowThreshold != databaseSlowQueryThreshold {
		t.Fatalf("slow threshold = %s, want %s", databaseLogger.config.SlowThreshold, databaseSlowQueryThreshold)
	}
	if !databaseLogger.config.IgnoreRecordNotFoundError {
		t.Fatal("RecordNotFound database error logs must remain suppressed")
	}
	if !databaseLogger.config.ParameterizedQueries {
		t.Fatal("database query logs must remain parameterized")
	}
}

func TestGormZapLoggerTraceContract(t *testing.T) {
	startedAt := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	databaseLogger, output := newJSONDatabaseLogger(t, databaseLoggerConfig())
	databaseLogger.now = func() time.Time { return startedAt.Add(databaseSlowQueryThreshold + time.Millisecond) }

	databaseLogger.Trace(context.Background(), startedAt, func() (string, int64) {
		return "SELECT * FROM scan_task WHERE id = ?", 3
	}, errors.New("driver error with secret value"))

	entry, raw := singleJSONLog(t, output)
	if got := entry["level"]; got != "error" {
		t.Fatalf("error query level = %v, want error", got)
	}
	if got := entry["msg"]; got != "database query failed" {
		t.Fatalf("error query message = %v", got)
	}
	if got := entry["db.query.text"]; got != "SELECT * FROM scan_task WHERE id = ?" {
		t.Fatalf("query text = %v", got)
	}
	if got := entry["db.rows_affected"]; got != float64(3) {
		t.Fatalf("rows affected = %v, want 3", got)
	}
	if got := entry["error"]; got != "database_query_failed" {
		t.Fatalf("safe error = %v, want database_query_failed", got)
	}
	if _, ok := entry["timestamp"]; !ok {
		t.Fatalf("missing shared timestamp envelope: %v", entry)
	}
	caller, _ := entry["caller"].(string)
	if !strings.Contains(caller, "database_test.go:") || strings.Contains(caller, "gorm_zap_logger.go:") {
		t.Fatalf("query caller = %q, want test query call site", caller)
	}
	if strings.Count(raw, "\"caller\":") != 1 {
		t.Fatalf("log must have one top-level caller, got %q", raw)
	}

	output.Reset()
	databaseLogger.now = func() time.Time { return startedAt.Add(databaseSlowQueryThreshold) }
	databaseLogger.Trace(context.Background(), startedAt, func() (string, int64) {
		return "SELECT 1", -1
	}, nil)
	if output.Len() != 0 {
		t.Fatalf("query at 200ms boundary should not be slow: %q", output.String())
	}

	output.Reset()
	databaseLogger.now = func() time.Time { return startedAt.Add(databaseSlowQueryThreshold + time.Millisecond) }
	databaseLogger.Trace(context.Background(), startedAt, func() (string, int64) {
		return "SELECT 1", -1
	}, nil)
	entry, _ = singleJSONLog(t, output)
	if got := entry["level"]; got != "warn" {
		t.Fatalf("slow query level = %v, want warn", got)
	}
	if _, ok := entry["db.rows_affected"]; ok {
		t.Fatalf("unknown row count must be omitted: %v", entry)
	}
}

func TestGormZapLoggerInfoAndDirectMethodsHonorLogMode(t *testing.T) {
	startedAt := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	config := databaseLoggerConfig()
	config.LogLevel = logger.Info
	databaseLogger, output := newJSONDatabaseLogger(t, config)
	databaseLogger.now = func() time.Time { return startedAt.Add(time.Millisecond) }

	databaseLogger.Trace(context.Background(), startedAt, func() (string, int64) {
		return "SELECT 1", 0
	}, nil)
	databaseLogger.Info(context.Background(), "gorm info %d", 1)
	databaseLogger.Warn(context.Background(), "gorm warn %d", 2)
	databaseLogger.Error(context.Background(), "gorm error %d", 3)

	entries := jsonLogs(t, output)
	if len(entries) != 4 {
		t.Fatalf("logged entries = %d, want 4", len(entries))
	}
	if got := entries[0]["level"]; got != "info" {
		t.Fatalf("ordinary Info trace level = %v, want info", got)
	}
	for index, want := range []string{"info", "warn", "error"} {
		if got := entries[index+1]["level"]; got != want {
			t.Fatalf("direct method %d level = %v, want %s", index, got, want)
		}
	}

	output.Reset()
	warnLogger, ok := databaseLogger.LogMode(logger.Warn).(*gormZapLogger)
	if !ok {
		t.Fatalf("LogMode result = %T, want *gormZapLogger", databaseLogger.LogMode(logger.Warn))
	}
	warnLogger.Info(context.Background(), "suppressed")
	warnLogger.Trace(context.Background(), startedAt, func() (string, int64) {
		return "SELECT 1", 0
	}, nil)
	if output.Len() != 0 {
		t.Fatalf("Warn mode emitted Info logs: %q", output.String())
	}
}

func TestGormZapLoggerSuppressesRecordNotFoundErrors(t *testing.T) {
	databaseLogger, output := newJSONDatabaseLogger(t, databaseLoggerConfig())
	databaseLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT * FROM scan_task", 0
	}, gorm.ErrRecordNotFound)

	if output.Len() != 0 {
		t.Fatalf("RecordNotFound emitted a database error log: %q", output.String())
	}
}

func TestGormZapLoggerDoesNotLeakBoundSecrets(t *testing.T) {
	databaseLogger, output := newJSONDatabaseLogger(t, databaseLoggerConfig())
	const secret = "lunafox-secret-2b1b601f"
	derivedSecret := strings.ToUpper(secret)

	filteredSQL, params := databaseLogger.ParamsFilter(context.Background(), "SELECT * FROM registration_token WHERE token = ?", secret)
	if filteredSQL != "SELECT * FROM registration_token WHERE token = ?" || params != nil {
		t.Fatalf("parameter filter = (%q, %v), want original SQL and nil params", filteredSQL, params)
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: databaseLogger})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec("SELECT * FROM missing_table WHERE token = ?", secret).Error; err == nil {
		t.Fatal("expected failed query")
	}
	entries := jsonLogs(t, output)
	if len(entries) != 1 {
		t.Fatalf("GORM query logs = %d, want 1; output=%q", len(entries), output.String())
	}
	caller, _ := entries[0]["caller"].(string)
	if !strings.Contains(caller, "database_test.go:") || strings.Contains(caller, "gorm_zap_logger.go:") {
		t.Fatalf("GORM query caller = %q, want test query call site", caller)
	}
	databaseLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT * FROM registration_token WHERE token = ?", 0
	}, fmt.Errorf("driver detail contains %s", derivedSecret))

	raw := output.String()
	if !strings.Contains(raw, "token = ?") {
		t.Fatalf("expected parameterized query text, got %q", raw)
	}
	if strings.Contains(raw, secret) || strings.Contains(raw, derivedSecret) {
		t.Fatalf("database log leaked bound or derived secret: %q", raw)
	}
}

func newJSONDatabaseLogger(t *testing.T, config logger.Config) (*gormZapLogger, *bytes.Buffer) {
	t.Helper()
	var output bytes.Buffer
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(&output), zapcore.DebugLevel)
	return newGormZapLogger(zap.New(core, zap.AddCaller()), config), &output
}

func singleJSONLog(t *testing.T, output *bytes.Buffer) (map[string]interface{}, string) {
	t.Helper()
	entries := jsonLogs(t, output)
	if len(entries) != 1 {
		t.Fatalf("log entry count = %d, want 1; output=%q", len(entries), output.String())
	}
	return entries[0], strings.TrimSpace(output.String())
}

func jsonLogs(t *testing.T, output *bytes.Buffer) []map[string]interface{} {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	entries := make([]map[string]interface{}, 0, len(lines))
	for _, line := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("invalid JSON log line %q: %v", line, err)
		}
		entries = append(entries, entry)
	}
	return entries
}
