package pkg

import (
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// LogFieldRequestID is the semantic request correlation field name.
	LogFieldRequestID = "request.id"
)

var (
	// Logger is the global logger instance.
	Logger *zap.Logger
	// Sugar is the sugared logger for convenience.
	Sugar *zap.SugaredLogger

	loggerMu sync.RWMutex
)

func ensureLogger() *zap.Logger {
	loggerMu.RLock()
	if Logger != nil {
		logger := Logger
		loggerMu.RUnlock()
		return logger
	}
	loggerMu.RUnlock()

	loggerMu.Lock()
	defer loggerMu.Unlock()
	if Logger != nil {
		return Logger
	}
	Logger = zap.NewNop()
	Sugar = Logger.Sugar()
	return Logger
}

func setLogger(logger *zap.Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	Logger = logger
	if logger == nil {
		Sugar = nil
		return
	}
	Sugar = logger.Sugar()
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level string
}

// InitLogger initializes the global logger.
func InitLogger(cfg *LogConfig) error {
	level := parseLogLevel(cfg.Level)
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return err
	}

	setLogger(logger)
	return nil
}

// parseLogLevel converts string level to zapcore.Level.
func parseLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// Sync flushes any buffered log entries.
func Sync() {
	loggerMu.RLock()
	logger := Logger
	loggerMu.RUnlock()
	if logger == nil {
		return
	}
	_ = logger.Sync()
}

// Debug logs a debug message.
func Debug(msg string, fields ...zap.Field) {
	ensureLogger().Debug(msg, fields...)
}

// Info logs an info message.
func Info(msg string, fields ...zap.Field) {
	ensureLogger().Info(msg, fields...)
}

// Warn logs a warning message.
func Warn(msg string, fields ...zap.Field) {
	ensureLogger().Warn(msg, fields...)
}

// Error logs an error message.
func Error(msg string, fields ...zap.Field) {
	ensureLogger().Error(msg, fields...)
}

// Fatal logs a fatal message and exits.
func Fatal(msg string, fields ...zap.Field) {
	ensureLogger().Fatal(msg, fields...)
}

// With creates a child logger with additional fields.
func With(fields ...zap.Field) *zap.Logger {
	return ensureLogger().With(fields...)
}

// RequestIDField returns the semantic request ID field.
func RequestIDField(requestID string) zap.Field {
	return zap.String(LogFieldRequestID, requestID)
}

// WithRequestID creates a logger with request ID field.
func WithRequestID(requestID string) *zap.Logger {
	return ensureLogger().With(RequestIDField(requestID))
}

// NewNopLogger returns a no-op logger for testing.
func NewNopLogger() *zap.Logger {
	return zap.NewNop()
}

// InitTestLogger initializes a test logger that writes to stdout.
func InitTestLogger() {
	setLogger(zap.NewExample())
}

// InitDefaultLogger initializes logger with default settings.
func InitDefaultLogger() error {
	return InitLogger(&LogConfig{
		Level: os.Getenv("LOG_LEVEL"),
	})
}
