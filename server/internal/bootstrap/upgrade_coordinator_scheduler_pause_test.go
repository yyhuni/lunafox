package bootstrap

import (
	"context"
	"testing"
	"time"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scheduledscanapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
	"go.uber.org/zap"
)

// A full Upgrade Operation pauses the scheduler before cancelling active
// scans, assuming the host handoff will replace this process with an unpaused
// successor. When preparation fails after the pause, the process keeps
// serving, so the scheduler must resume; an un-resumed in-memory pause kills
// every Schedule — including per-minute ones — until a manual restart.
func TestUpgradePreparationFailureResumesScheduledScanScheduler(t *testing.T) {
	clock := &pauseReproClock{now: time.Date(2026, 10, 4, 8, 0, 0, 0, time.UTC)}
	repository := &pauseReproRepository{}
	controller := scheduledscanapp.NewSchedulerController(repository, &pauseReproDispatcher{}).
		WithRuntime(clock, pauseReproWaiter{}, pauseReproLogger{logger: zap.NewNop()})

	baseline := controller.RunPass(context.Background())
	if baseline.Materialized != 1 || controller.IsPaused() {
		t.Fatalf("baseline pass = %+v paused=%t; want one due schedule materialized while unpaused", baseline, controller.IsPaused())
	}
	repository.reset()

	// A zero-value ScanFacade fails StopAllActiveForUpgrade, matching the
	// production path where markPreparationFailed ends the operation failed
	// while the server process keeps running.
	coordinator := newUpgradePreDispatchCoordinator(&scanapp.ScanFacade{}, controller)
	if _, err := coordinator.PrepareForUpgrade(context.Background(), "pause-repro"); err == nil {
		t.Fatal("PrepareForUpgrade() with an unconfigured scan facade must fail to model a failed preparation")
	}

	passesWithError := 0
	for minute := 1; minute <= 30; minute++ {
		clock.Advance(time.Minute)
		if pass := controller.RunPass(context.Background()); pass.Err != nil {
			passesWithError++
		}
	}
	if controller.IsPaused() {
		t.Fatal("scheduler is still paused after failed upgrade preparation; scheduled scans stay dead until process restart")
	}
	if materialized := repository.materializedCount(); materialized != 30 {
		t.Fatalf("per-minute schedule materialized %d/30 idle passes after failed upgrade preparation (error passes: %d)", materialized, passesWithError)
	}
}

type pauseReproClock struct{ now time.Time }

func (clock *pauseReproClock) Now() time.Time { return clock.now }

func (clock *pauseReproClock) Advance(duration time.Duration) { clock.now = clock.now.Add(duration) }

type pauseReproRepository struct {
	materialized int
}

func (repository *pauseReproRepository) ListDueSchedules(_ context.Context, evaluationAt time.Time) ([]scheduledscanapp.DueSchedule, error) {
	return []scheduledscanapp.DueSchedule{{ID: 1, NextRunTime: evaluationAt.Add(-time.Minute)}}, nil
}

func (repository *pauseReproRepository) MaterializeDue(context.Context, int, time.Time) (bool, error) {
	repository.materialized++
	return true, nil
}

func (repository *pauseReproRepository) SelectAttemptCandidate(context.Context) (*scheduledscanapp.OccurrenceCandidate, error) {
	return nil, nil
}

func (repository *pauseReproRepository) StartAttempt(context.Context, scheduledscanapp.OccurrenceCandidate, time.Time) (*scheduledscanapp.FrozenDispatchInput, error) {
	return nil, nil
}

func (repository *pauseReproRepository) RecordOutcome(context.Context, int64, scheduledscanapp.HandoffOutcome, time.Time) (bool, error) {
	return false, nil
}

func (repository *pauseReproRepository) EarliestRetryDeadline(context.Context) (*time.Time, error) {
	return nil, nil
}

func (repository *pauseReproRepository) reset() {
	repository.materialized = 0
}

func (repository *pauseReproRepository) materializedCount() int {
	return repository.materialized
}

type pauseReproDispatcher struct{}

func (dispatcher *pauseReproDispatcher) Dispatch(context.Context, scheduledscanapp.FrozenDispatchInput) (*scanapp.BatchScanResult, error) {
	return &scanapp.BatchScanResult{CreatedCount: 1}, nil
}

type pauseReproWaiter struct{}

func (waiter pauseReproWaiter) Wait(context.Context, time.Duration) error { return nil }

type pauseReproLogger struct{ logger *zap.Logger }

func (logger pauseReproLogger) Debug(message string, fields ...zap.Field) {
	logger.logger.Debug(message, fields...)
}

func (logger pauseReproLogger) Info(message string, fields ...zap.Field) {
	logger.logger.Info(message, fields...)
}

func (logger pauseReproLogger) Error(message string, fields ...zap.Field) {
	logger.logger.Error(message, fields...)
}
