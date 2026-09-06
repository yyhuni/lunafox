package application

import (
	"context"
	"errors"
	"time"

	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

type RuntimeClock interface {
	Now() time.Time
}

type RuntimeWaiter interface {
	Wait(ctx context.Context, duration time.Duration) error
}

type EventLogger interface {
	Debug(message string, fields ...zap.Field)
	Info(message string, fields ...zap.Field)
	Error(message string, fields ...zap.Field)
}

type systemRuntimeClock struct{}

func (systemRuntimeClock) Now() time.Time { return time.Now() }

type systemRuntimeWaiter struct{}

func (systemRuntimeWaiter) Wait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type packageEventLogger struct{}

func (packageEventLogger) Debug(message string, fields ...zap.Field) { pkg.Debug(message, fields...) }
func (packageEventLogger) Info(message string, fields ...zap.Field)  { pkg.Info(message, fields...) }
func (packageEventLogger) Error(message string, fields ...zap.Field) { pkg.Error(message, fields...) }

func runtimeErrorKind(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "operation_failed"
	}
}
