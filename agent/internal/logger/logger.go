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
	logger, err := buildLogger(level, os.Getenv("ENV"), zapcore.Lock(os.Stdout), zapcore.Lock(os.Stderr))
	if err != nil {
		Log = zap.NewNop()
		return err
	}
	Log = logger
	return nil
}

func buildLogger(level, env string, stdout, stderr zapcore.WriteSyncer) (*zap.Logger, error) {
	level = strings.TrimSpace(level)
	if level == "" {
		level = "info"
	}

	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	isDev := strings.EqualFold(env, "development")
	stdoutLevel := zap.LevelEnablerFunc(func(entryLevel zapcore.Level) bool {
		return entryLevel >= zapLevel && entryLevel < zapcore.WarnLevel
	})
	stderrLevel := zap.LevelEnablerFunc(func(entryLevel zapcore.Level) bool {
		return entryLevel >= zapLevel && entryLevel >= zapcore.WarnLevel
	})

	core := zapcore.NewTee(
		zapcore.NewCore(newEncoder(isDev), stdout, stdoutLevel),
		zapcore.NewCore(newEncoder(isDev), stderr, stderrLevel),
	)

	options := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	}
	if isDev {
		options = append(options, zap.Development())
	}

	return zap.New(core, options...), nil
}

func newEncoder(isDev bool) zapcore.Encoder {
	if isDev {
		config := zap.NewDevelopmentEncoderConfig()
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return zapcore.NewConsoleEncoder(config)
	}
	return zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
}

// Sync flushes any buffered log entries.
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
