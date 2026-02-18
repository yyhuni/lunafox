package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the shared agent logger. Defaults to a no-op logger until initialized.
var Log = zap.NewNop()

// Init configures the logger using the provided level and ENV.
func Init(level string) error {
	level = strings.TrimSpace(level)
	if level == "" {
		level = "info"
	}

	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	isDev := strings.EqualFold(os.Getenv("ENV"), "development")
	var config zap.Config
	if isDev {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	logger, err := config.Build()
	if err != nil {
		Log = zap.NewNop()
		return err
	}
	Log = logger
	return nil
}

// Sync flushes any buffered log entries.
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
