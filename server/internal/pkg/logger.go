package pkg

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Logger is the global logger instance
	Logger *zap.Logger
	// Sugar is the sugared logger for convenience
	Sugar *zap.SugaredLogger
)

func ensureLogger() {
	if Logger != nil {
		return
	}
	Logger = zap.NewNop()
	Sugar = Logger.Sugar()
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string
	Format string
}

// InitLogger initializes the global logger
func InitLogger(cfg *LogConfig) error {
	level := parseLogLevel(cfg.Level)

	var config zap.Config
	if strings.ToLower(cfg.Format) == "json" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	config.Level = zap.NewAtomicLevelAt(level)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var err error
	Logger, err = config.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return err
	}

	Sugar = Logger.Sugar()
	return nil
}

// parseLogLevel converts string level to zapcore.Level
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

// Sync flushes any buffered log entries
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	ensureLogger()
	Logger.Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	ensureLogger()
	Logger.Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	ensureLogger()
	Logger.Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	ensureLogger()
	Logger.Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	ensureLogger()
	Logger.Fatal(msg, fields...)
}

// With creates a child logger with additional fields
func With(fields ...zap.Field) *zap.Logger {
	ensureLogger()
	return Logger.With(fields...)
}

// WithRequestID creates a logger with request ID field
func WithRequestID(requestID string) *zap.Logger {
	ensureLogger()
	return Logger.With(zap.String("requestId", requestID))
}

// NewNopLogger returns a no-op logger for testing
func NewNopLogger() *zap.Logger {
	return zap.NewNop()
}

// InitTestLogger initializes a test logger that writes to stdout
func InitTestLogger() {
	Logger = zap.NewExample()
	Sugar = Logger.Sugar()
}

// InitDefaultLogger initializes logger with default settings
func InitDefaultLogger() error {
	return InitLogger(&LogConfig{
		Level:  os.Getenv("LOG_LEVEL"),
		Format: os.Getenv("LOG_FORMAT"),
	})
}
