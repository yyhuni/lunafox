package database

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const databaseSlowQueryThreshold = 200 * time.Millisecond

type gormZapLogger struct {
	logger *zap.Logger
	config logger.Config
	now    func() time.Time
}

func newGormZapLogger(zapLogger *zap.Logger, config logger.Config) *gormZapLogger {
	if zapLogger == nil {
		zapLogger = zap.NewNop()
	}
	return &gormZapLogger{
		logger: zapLogger,
		config: config,
		now:    time.Now,
	}
}

func (l *gormZapLogger) LogMode(level logger.LogLevel) logger.Interface {
	copy := *l
	copy.config.LogLevel = level
	return &copy
}

func (l *gormZapLogger) Info(_ context.Context, message string, data ...interface{}) {
	if l.config.LogLevel < logger.Info {
		return
	}
	l.log(zap.InfoLevel, formatGormLogMessage(message, data...), databaseLogCaller())
}

func (l *gormZapLogger) Warn(_ context.Context, message string, data ...interface{}) {
	if l.config.LogLevel < logger.Warn {
		return
	}
	l.log(zap.WarnLevel, formatGormLogMessage(message, data...), databaseLogCaller())
}

func (l *gormZapLogger) Error(_ context.Context, message string, data ...interface{}) {
	if l.config.LogLevel < logger.Error {
		return
	}
	l.log(zap.ErrorLevel, formatGormLogMessage(message, data...), databaseLogCaller())
}

func (l *gormZapLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.config.LogLevel <= logger.Silent {
		return
	}

	elapsed := l.now().Sub(begin)
	switch {
	case err != nil && l.config.LogLevel >= logger.Error && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.config.IgnoreRecordNotFoundError):
		sql, rowsAffected := fc()
		l.logQuery(zap.ErrorLevel, "database query failed", databaseLogCaller(), sql, elapsed, rowsAffected, zap.String("error", databaseErrorKind(err)))
	case elapsed > l.config.SlowThreshold && l.config.SlowThreshold != 0 && l.config.LogLevel >= logger.Warn:
		sql, rowsAffected := fc()
		l.logQuery(zap.WarnLevel, "database query slow", databaseLogCaller(), sql, elapsed, rowsAffected)
	case l.config.LogLevel == logger.Info:
		sql, rowsAffected := fc()
		l.logQuery(zap.InfoLevel, "database query completed", databaseLogCaller(), sql, elapsed, rowsAffected)
	}
}

// ParamsFilter keeps values out of the SQL GORM passes to Trace. The adapter
// never receives a rendered query that could expose a bound secret.
func (l *gormZapLogger) ParamsFilter(_ context.Context, sql string, _ ...interface{}) (string, []interface{}) {
	return sql, nil
}

func (l *gormZapLogger) logQuery(level zapcore.Level, message, caller, sql string, elapsed time.Duration, rowsAffected int64, extra ...zap.Field) {
	fields := []zap.Field{
		zap.String("db.query.text", sql),
		zap.Float64("db.query.duration_ms", float64(elapsed)/float64(time.Millisecond)),
	}
	if rowsAffected >= 0 {
		fields = append(fields, zap.Int64("db.rows_affected", rowsAffected))
	}
	fields = append(fields, extra...)
	l.log(level, message, caller, fields...)
}

func (l *gormZapLogger) log(level zapcore.Level, message, caller string, fields ...zap.Field) {
	fields = append([]zap.Field{zap.String("caller", caller)}, fields...)
	entryLogger := l.logger.WithOptions(zap.WithCaller(false))
	switch level {
	case zap.DebugLevel:
		entryLogger.Debug(message, fields...)
	case zap.InfoLevel:
		entryLogger.Info(message, fields...)
	case zap.WarnLevel:
		entryLogger.Warn(message, fields...)
	default:
		entryLogger.Error(message, fields...)
	}
}

func formatGormLogMessage(message string, data ...interface{}) string {
	if len(data) == 0 {
		return message
	}
	return fmt.Sprintf(message, data...)
}

func databaseErrorKind(err error) string {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return "record_not_found"
	case errors.Is(err, context.Canceled):
		return "context_cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "context_deadline_exceeded"
	default:
		return "database_query_failed"
	}
}

func databaseLogCaller() string {
	for skip := 2; skip < 32; skip++ {
		_, file, line, ok := runtime.Caller(skip)
		if !ok {
			return ""
		}
		if isDatabaseLoggerFrame(file) || isGORMFrame(file) {
			continue
		}
		return fmt.Sprintf("%s:%d", file, line)
	}
	return ""
}

func isDatabaseLoggerFrame(file string) bool {
	return filepath.Base(file) == "gorm_zap_logger.go"
}

func isGORMFrame(file string) bool {
	return strings.Contains(filepath.ToSlash(file), "/gorm.io/gorm@")
}
