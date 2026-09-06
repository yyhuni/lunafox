package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type retentionRepositoryStub struct {
	counts  []int64
	errors  []error
	err     error
	calls   int
	cutoffs []time.Time
	called  chan struct{}
}

func (stub *retentionRepositoryStub) DeleteOccurrenceBatch(_ context.Context, cutoff time.Time) (int64, error) {
	stub.cutoffs = append(stub.cutoffs, cutoff)
	index := stub.calls
	stub.calls++
	if stub.called != nil {
		select {
		case stub.called <- struct{}{}:
		default:
		}
	}
	count := int64(0)
	if index < len(stub.counts) {
		count = stub.counts[index]
	}
	if index < len(stub.errors) && stub.errors[index] != nil {
		return count, stub.errors[index]
	}
	if stub.err != nil {
		return count, stub.err
	}
	return count, nil
}

type retentionCycleWaiter struct {
	cancel    context.CancelFunc
	durations []time.Duration
}

func (waiter *retentionCycleWaiter) Wait(ctx context.Context, duration time.Duration) error {
	waiter.durations = append(waiter.durations, duration)
	if len(waiter.durations) == 1 {
		return nil
	}
	waiter.cancel()
	return ctx.Err()
}

type blockingRetentionRepository struct{}

func (blockingRetentionRepository) DeleteOccurrenceBatch(ctx context.Context, _ time.Time) (int64, error) {
	<-ctx.Done()
	return 0, ctx.Err()
}

func TestOccurrenceRetentionRunUsesSevenDayBoundaryAndStopsEarly(t *testing.T) {
	startedAt := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	repository := &retentionRepositoryStub{counts: []int64{1000, 1}}
	core, logs := observer.New(zap.DebugLevel)
	job := NewOccurrenceRetentionJob(repository).WithRuntime(
		&sequenceClock{times: []time.Time{startedAt, startedAt.Add(time.Second)}},
		&cancelingWaiter{},
		zapLoggerAdapter{logger: zap.New(core)},
	)

	deleted, err := job.RunOnce(context.Background())
	if err != nil || deleted != 1001 || repository.calls != 2 {
		t.Fatalf("RunOnce() = %d, %v calls=%d", deleted, err, repository.calls)
	}
	wantCutoff := startedAt.Add(-7 * 24 * time.Hour)
	if len(repository.cutoffs) != 2 || !repository.cutoffs[0].Equal(wantCutoff) {
		t.Fatalf("retention cutoffs = %+v, want %s", repository.cutoffs, wantCutoff)
	}
	if logs.FilterMessage("Scheduled scan occurrence retention completed").Len() != 1 {
		t.Fatal("positive successful run must emit exactly one Info")
	}
	fields := logs.FilterMessage("Scheduled scan occurrence retention completed").All()[0].ContextMap()
	if fmt.Sprint(fields["scheduled_scan.occurrence.deleted_count"]) != "1001" || fields["outcome"] != "completed" {
		t.Fatalf("retention success fields = %+v", fields)
	}
}

func TestOccurrenceRetentionRunCapsAtOneHundredThousandRows(t *testing.T) {
	counts := make([]int64, occurrenceRetentionBatches+1)
	for index := range counts {
		counts[index] = OccurrenceDeleteBatchSize
	}
	repository := &retentionRepositoryStub{counts: counts}
	job := NewOccurrenceRetentionJob(repository).WithRuntime(
		&sequenceClock{last: time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)},
		&cancelingWaiter{}, zapLoggerAdapter{logger: zap.NewNop()},
	)

	deleted, err := job.RunOnce(context.Background())
	if err != nil || deleted != 100000 || repository.calls != 100 {
		t.Fatalf("RunOnce() = %d, %v calls=%d", deleted, err, repository.calls)
	}
}

func TestOccurrenceRetentionFailureLogsNoSuccessAndDoesNotRetryInRun(t *testing.T) {
	wantErr := errors.New("database unavailable")
	repository := &retentionRepositoryStub{err: wantErr}
	core, logs := observer.New(zap.DebugLevel)
	job := NewOccurrenceRetentionJob(repository).WithRuntime(
		&sequenceClock{last: time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)},
		&cancelingWaiter{}, zapLoggerAdapter{logger: zap.New(core)},
	)

	deleted, err := job.RunOnce(context.Background())
	if !errors.Is(err, wantErr) || deleted != 0 || repository.calls != 1 {
		t.Fatalf("RunOnce() = %d, %v calls=%d", deleted, err, repository.calls)
	}
	if logs.FilterMessage("Scheduled scan occurrence retention completed").Len() != 0 || logs.FilterMessage("Scheduled scan occurrence retention failed").Len() != 1 {
		t.Fatalf("retention logs = %+v", logs.All())
	}
	fields := logs.FilterMessage("Scheduled scan occurrence retention failed").All()[0].ContextMap()
	if fields["error_kind"] != "operation_failed" || fmt.Sprint(fields["scheduled_scan.occurrence.deleted_count"]) != "0" {
		t.Fatalf("retention failure fields = %+v", fields)
	}
	if _, exposed := fields["error"]; exposed {
		t.Fatalf("retention log exposed a raw error: %+v", fields)
	}
}

func TestOccurrenceRetentionZeroDeleteSuccessIsSilent(t *testing.T) {
	repository := &retentionRepositoryStub{counts: []int64{0}}
	core, logs := observer.New(zap.DebugLevel)
	job := NewOccurrenceRetentionJob(repository).WithRuntime(
		&sequenceClock{last: time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)},
		&cancelingWaiter{}, zapLoggerAdapter{logger: zap.New(core)},
	)

	deleted, err := job.RunOnce(context.Background())
	if err != nil || deleted != 0 || repository.calls != 1 {
		t.Fatalf("RunOnce() = %d, %v calls=%d", deleted, err, repository.calls)
	}
	if logs.FilterLevelExact(zap.InfoLevel).Len() != 0 || logs.FilterLevelExact(zap.ErrorLevel).Len() != 0 {
		t.Fatalf("zero-delete success emitted outcome logs: %+v", logs.All())
	}
}

func TestOccurrenceRetentionPreservesCommittedBatchesWhenLaterBatchFails(t *testing.T) {
	wantErr := errors.New("second batch failed")
	repository := &retentionRepositoryStub{
		counts: []int64{OccurrenceDeleteBatchSize, 0},
		errors: []error{nil, wantErr},
	}
	core, logs := observer.New(zap.DebugLevel)
	job := NewOccurrenceRetentionJob(repository).WithRuntime(
		&sequenceClock{last: time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)},
		&cancelingWaiter{}, zapLoggerAdapter{logger: zap.New(core)},
	)

	deleted, err := job.RunOnce(context.Background())
	if !errors.Is(err, wantErr) || deleted != OccurrenceDeleteBatchSize || repository.calls != 2 {
		t.Fatalf("RunOnce() = %d, %v calls=%d", deleted, err, repository.calls)
	}
	if logs.FilterMessage("Scheduled scan occurrence retention failed").Len() != 1 || logs.FilterLevelExact(zap.InfoLevel).Len() != 0 {
		t.Fatalf("partial-failure logs = %+v", logs.All())
	}
}

func TestOccurrenceRetentionHonorsParentDeadline(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	job := NewOccurrenceRetentionJob(blockingRetentionRepository{}).WithRuntime(
		&sequenceClock{last: time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)},
		&cancelingWaiter{}, zapLoggerAdapter{logger: zap.New(core)},
	)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	deleted, err := job.RunOnce(ctx)
	if !errors.Is(err, context.DeadlineExceeded) || deleted != 0 {
		t.Fatalf("RunOnce() = %d, %v; want parent deadline", deleted, err)
	}
	if logs.FilterMessage("Scheduled scan occurrence retention failed").Len() != 1 || logs.FilterLevelExact(zap.InfoLevel).Len() != 0 {
		t.Fatalf("deadline logs = %+v", logs.All())
	}
}

func TestOccurrenceRetentionStartRunsImmediatelyThenRetriesOnlyAfterHourlyWait(t *testing.T) {
	wantErr := errors.New("startup cleanup failed")
	repository := &retentionRepositoryStub{
		counts: []int64{0, 0}, errors: []error{wantErr, nil}, called: make(chan struct{}, 2),
	}
	ctx, cancel := context.WithCancel(context.Background())
	waiter := &retentionCycleWaiter{cancel: cancel}
	job := NewOccurrenceRetentionJob(repository).WithRuntime(
		&sequenceClock{last: time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)}, waiter, zapLoggerAdapter{logger: zap.NewNop()},
	)

	startedAt := time.Now()
	job.Start(ctx)
	if time.Since(startedAt) > 50*time.Millisecond {
		t.Fatal("Start() blocked on startup cleanup")
	}
	select {
	case <-repository.called:
	case <-time.After(time.Second):
		t.Fatal("startup cleanup did not run immediately")
	}
	select {
	case <-job.Done():
	case <-time.After(time.Second):
		t.Fatal("retention job did not finish test cycles")
	}
	if repository.calls != 2 || len(waiter.durations) != 2 {
		t.Fatalf("retention cycles = calls %d waits %+v", repository.calls, waiter.durations)
	}
	for _, duration := range waiter.durations {
		if duration != occurrenceRetentionInterval {
			t.Fatalf("retention wait = %s, want %s", duration, occurrenceRetentionInterval)
		}
	}
}

func TestOccurrenceRetentionFailureDoesNotStopSiblingScheduler(t *testing.T) {
	rootCtx, cancel := context.WithCancel(context.Background())
	retentionWaiter := &blockingWaiter{entered: make(chan time.Duration, 1)}
	retention := NewOccurrenceRetentionJob(&retentionRepositoryStub{err: errors.New("retention failed")}).WithRuntime(
		&sequenceClock{last: testTime()}, retentionWaiter, zapLoggerAdapter{logger: zap.NewNop()},
	)
	schedulerRepository := &schedulerRepositoryStub{passCalls: make(chan struct{}, 1)}
	schedulerWaiter := &blockingWaiter{entered: make(chan time.Duration, 1)}
	scheduler := NewSchedulerController(schedulerRepository, &dispatcherStub{}).WithRuntime(
		&sequenceClock{last: testTime()}, schedulerWaiter, zapLoggerAdapter{logger: zap.NewNop()},
	)

	retention.Start(rootCtx)
	select {
	case <-retentionWaiter.entered:
	case <-time.After(time.Second):
		t.Fatal("retention failure did not reach the normal hourly wait")
	}
	if rootCtx.Err() != nil {
		t.Fatalf("retention failure canceled the shared root: %v", rootCtx.Err())
	}
	scheduler.Start(rootCtx)
	select {
	case <-schedulerRepository.passCalls:
	case <-time.After(time.Second):
		t.Fatal("sibling scheduler did not run after retention failure")
	}

	cancel()
	select {
	case <-retention.Done():
	case <-time.After(time.Second):
		t.Fatal("retention did not stop on shared cancellation")
	}
	select {
	case <-scheduler.Done():
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop on shared cancellation")
	}
}
