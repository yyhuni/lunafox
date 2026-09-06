package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type schedulerRepositoryStub struct {
	mu               sync.Mutex
	due              []DueSchedule
	listErr          error
	materializeErr   error
	materialized     int
	candidate        *OccurrenceCandidate
	startInput       *FrozenDispatchInput
	startCalls       int
	startErr         error
	recordedOutcomes []HandoffOutcome
	recordErr        error
	recorded         bool
	honorRecordCtx   bool
	passCalls        chan struct{}
}

func (stub *schedulerRepositoryStub) ListDueSchedules(context.Context, time.Time) ([]DueSchedule, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.passCalls != nil {
		select {
		case stub.passCalls <- struct{}{}:
		default:
		}
	}
	return append([]DueSchedule(nil), stub.due...), stub.listErr
}

func (stub *schedulerRepositoryStub) MaterializeDue(context.Context, int, time.Time) (bool, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	stub.materialized++
	return true, stub.materializeErr
}

func (stub *schedulerRepositoryStub) SelectAttemptCandidate(context.Context) (*OccurrenceCandidate, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.candidate == nil {
		return nil, nil
	}
	copy := *stub.candidate
	return &copy, nil
}

func (stub *schedulerRepositoryStub) StartAttempt(context.Context, OccurrenceCandidate, time.Time) (*FrozenDispatchInput, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	stub.startCalls++
	if stub.startErr != nil {
		return nil, stub.startErr
	}
	if stub.startInput == nil {
		return nil, nil
	}
	copy := *stub.startInput
	stub.candidate = nil
	return &copy, nil
}

func (stub *schedulerRepositoryStub) RecordOutcome(ctx context.Context, _ int64, outcome HandoffOutcome, _ time.Time) (bool, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.honorRecordCtx && ctx.Err() != nil {
		return false, ctx.Err()
	}
	stub.recordedOutcomes = append(stub.recordedOutcomes, outcome)
	return stub.recorded, stub.recordErr
}

type dispatcherStub struct {
	result *scanapp.BatchScanResult
	err    error
	calls  int
}

type drainingSchedulerRepository struct {
	mu           sync.Mutex
	batches      [][]DueSchedule
	listCalls    int
	materialized int
}

func (repository *drainingSchedulerRepository) ListDueSchedules(context.Context, time.Time) ([]DueSchedule, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.listCalls++
	if len(repository.batches) == 0 {
		return nil, nil
	}
	batch := append([]DueSchedule(nil), repository.batches[0]...)
	repository.batches = repository.batches[1:]
	return batch, nil
}

func (repository *drainingSchedulerRepository) MaterializeDue(context.Context, int, time.Time) (bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.materialized++
	return true, nil
}

func (*drainingSchedulerRepository) SelectAttemptCandidate(context.Context) (*OccurrenceCandidate, error) {
	return nil, nil
}

func (*drainingSchedulerRepository) StartAttempt(context.Context, OccurrenceCandidate, time.Time) (*FrozenDispatchInput, error) {
	return nil, nil
}

func (*drainingSchedulerRepository) RecordOutcome(context.Context, int64, HandoffOutcome, time.Time) (bool, error) {
	return false, nil
}

func (repository *drainingSchedulerRepository) snapshot() (int, int) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return repository.listCalls, repository.materialized
}

type blockingWaiter struct{ entered chan time.Duration }

func (waiter *blockingWaiter) Wait(ctx context.Context, duration time.Duration) error {
	waiter.entered <- duration
	<-ctx.Done()
	return ctx.Err()
}

type deadlineCapturingDispatcher struct {
	deadline time.Time
	ok       bool
}

type cancelBlockingDispatcher struct {
	started chan struct{}
	calls   int
}

func (dispatcher *cancelBlockingDispatcher) Dispatch(ctx context.Context, _ FrozenDispatchInput) (*scanapp.BatchScanResult, error) {
	dispatcher.calls++
	select {
	case dispatcher.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

type queuedAttemptRepository struct {
	mu       sync.Mutex
	inputs   []FrozenDispatchInput
	recorded int
}

func (*queuedAttemptRepository) ListDueSchedules(context.Context, time.Time) ([]DueSchedule, error) {
	return nil, nil
}

func (*queuedAttemptRepository) MaterializeDue(context.Context, int, time.Time) (bool, error) {
	return false, nil
}

func (repository *queuedAttemptRepository) SelectAttemptCandidate(context.Context) (*OccurrenceCandidate, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if len(repository.inputs) == 0 {
		return nil, nil
	}
	input := repository.inputs[0]
	return &OccurrenceCandidate{ID: input.OccurrenceID, ScheduledScanID: input.ScheduledScanID, ScheduledFor: input.ScheduledFor}, nil
}

func (repository *queuedAttemptRepository) StartAttempt(context.Context, OccurrenceCandidate, time.Time) (*FrozenDispatchInput, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if len(repository.inputs) == 0 {
		return nil, nil
	}
	input := repository.inputs[0]
	repository.inputs = repository.inputs[1:]
	return &input, nil
}

func (repository *queuedAttemptRepository) RecordOutcome(context.Context, int64, HandoffOutcome, time.Time) (bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.recorded++
	return true, nil
}

func (dispatcher *deadlineCapturingDispatcher) Dispatch(ctx context.Context, _ FrozenDispatchInput) (*scanapp.BatchScanResult, error) {
	dispatcher.deadline, dispatcher.ok = ctx.Deadline()
	return &scanapp.BatchScanResult{CreatedCount: 1, Scans: []scanapp.QueryScan{{ID: 99}}}, nil
}

func (stub *dispatcherStub) Dispatch(context.Context, FrozenDispatchInput) (*scanapp.BatchScanResult, error) {
	stub.calls++
	return stub.result, stub.err
}

type sequenceClock struct {
	mu    sync.Mutex
	times []time.Time
	last  time.Time
}

func (clock *sequenceClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	if len(clock.times) > 0 {
		clock.last = clock.times[0]
		clock.times = clock.times[1:]
	}
	return clock.last
}

type cancelingWaiter struct {
	cancel    context.CancelFunc
	durations chan time.Duration
}

func (waiter *cancelingWaiter) Wait(ctx context.Context, duration time.Duration) error {
	if waiter.durations != nil {
		waiter.durations <- duration
	}
	if waiter.cancel != nil {
		waiter.cancel()
	}
	return ctx.Err()
}

type zapLoggerAdapter struct{ logger *zap.Logger }

func (adapter zapLoggerAdapter) Debug(message string, fields ...zap.Field) {
	adapter.logger.Debug(message, fields...)
}
func (adapter zapLoggerAdapter) Info(message string, fields ...zap.Field) {
	adapter.logger.Info(message, fields...)
}
func (adapter zapLoggerAdapter) Error(message string, fields ...zap.Field) {
	adapter.logger.Error(message, fields...)
}

func TestSchedulerPassMaterializesBatchAndDispatchesAtMostOneOccurrence(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	targetID := 7
	repository := &schedulerRepositoryStub{
		due:        []DueSchedule{{ID: 1}, {ID: 2}},
		candidate:  &OccurrenceCandidate{ID: 9, ScheduledScanID: 1, ScheduledFor: now},
		startInput: &FrozenDispatchInput{OccurrenceID: 9, ScheduledScanID: 1, ScheduledFor: now, ScanWorkflowID: "default", InputSource: scanapp.InputSourceScanSnapshot, TargetIDs: []int{targetID}, TargetScoped: true},
		recorded:   true,
	}
	dispatcher := &dispatcherStub{result: &scanapp.BatchScanResult{CreatedCount: 1, Scans: []scanapp.QueryScan{{ID: 99}}}}
	core, logs := observer.New(zap.DebugLevel)
	controller := NewSchedulerController(repository, dispatcher).WithRuntime(
		&sequenceClock{times: []time.Time{now, now.Add(time.Second), now.Add(2 * time.Second)}},
		&cancelingWaiter{},
		zapLoggerAdapter{logger: zap.New(core)},
	)

	result := controller.RunPass(context.Background())
	if result.Err != nil || result.Materialized != 2 || !result.Attempted || dispatcher.calls != 1 || repository.startCalls != 1 {
		t.Fatalf("RunPass() = %+v dispatch=%d starts=%d", result, dispatcher.calls, repository.startCalls)
	}
	if len(repository.recordedOutcomes) != 1 || repository.recordedOutcomes[0].Kind != HandoffCompleted {
		t.Fatalf("recorded outcomes = %+v", repository.recordedOutcomes)
	}
	if logs.FilterMessage("Scheduled scan handoff completed").Len() != 1 {
		t.Fatalf("success log count = %d", logs.FilterMessage("Scheduled scan handoff completed").Len())
	}
	fields := logs.FilterMessage("Scheduled scan handoff completed").All()[0].ContextMap()
	if fmt.Sprint(fields["scheduled_scan.id"]) != "1" || fmt.Sprint(fields["scheduled_scan.occurrence.id"]) != "9" || fields["outcome"] != "completed" {
		t.Fatalf("handoff success fields = %+v", fields)
	}
}

func TestSchedulerFailedHandoffStillCountsAsProgressWithoutRetry(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	targetID := 7
	repository := &schedulerRepositoryStub{
		candidate:  &OccurrenceCandidate{ID: 9, ScheduledScanID: 1, ScheduledFor: now},
		startInput: &FrozenDispatchInput{OccurrenceID: 9, ScheduledScanID: 1, ScheduledFor: now, ScanWorkflowID: "default", InputSource: scanapp.InputSourceScanSnapshot, TargetIDs: []int{targetID}, TargetScoped: true},
		recorded:   true,
	}
	dispatcher := &dispatcherStub{err: errors.New("private backend failure")}
	core, logs := observer.New(zap.DebugLevel)
	controller := NewSchedulerController(repository, dispatcher).WithRuntime(
		&sequenceClock{times: []time.Time{now, now, now}}, &cancelingWaiter{}, zapLoggerAdapter{logger: zap.New(core)},
	)

	result := controller.RunPass(context.Background())
	if result.Err != nil || !result.Progressed() || !result.Attempted || dispatcher.calls != 1 {
		t.Fatalf("RunPass() = %+v dispatch=%d", result, dispatcher.calls)
	}
	if len(repository.recordedOutcomes) != 1 || repository.recordedOutcomes[0].Kind != HandoffScanCreateFailed {
		t.Fatalf("recorded outcomes = %+v", repository.recordedOutcomes)
	}
	if logs.FilterMessage("Scheduled scan handoff did not complete").Len() != 1 {
		t.Fatalf("failure log count = %d", logs.FilterMessage("Scheduled scan handoff did not complete").Len())
	}
	fields := logs.FilterMessage("Scheduled scan handoff did not complete").All()[0].ContextMap()
	if fields["outcome"] != "scan_create_failed" {
		t.Fatalf("handoff failure fields = %+v", fields)
	}
	if strings.Contains(fmt.Sprint(fields), "private backend failure") {
		t.Fatalf("handoff failure log exposed raw error text: %+v", fields)
	}
}

func TestSchedulerStartIsNonBlockingRunsImmediatelyAndWaitsOneMinuteWhenIdle(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	ctx, cancel := context.WithCancel(context.Background())
	repository := &schedulerRepositoryStub{passCalls: make(chan struct{}, 1)}
	waits := make(chan time.Duration, 1)
	controller := NewSchedulerController(repository, &dispatcherStub{}).WithRuntime(
		&sequenceClock{last: now}, &cancelingWaiter{cancel: cancel, durations: waits}, zapLoggerAdapter{logger: zap.NewNop()},
	)

	startedAt := time.Now()
	controller.Start(ctx)
	if time.Since(startedAt) > 50*time.Millisecond {
		t.Fatal("Start() blocked on the initial pass")
	}
	select {
	case <-repository.passCalls:
	case <-time.After(time.Second):
		t.Fatal("initial pass did not run immediately")
	}
	select {
	case duration := <-waits:
		if duration != time.Minute {
			t.Fatalf("idle wait = %s, want one minute", duration)
		}
	case <-time.After(time.Second):
		t.Fatal("idle pass did not enter normal wait")
	}
	select {
	case <-controller.Done():
	case <-time.After(time.Second):
		t.Fatal("controller did not exit after cancellation")
	}
}

func TestSchedulerControllerDrainsCommittedBatchesBeforeWaiting(t *testing.T) {
	firstBatch := make([]DueSchedule, DueScheduleBatchSize)
	for index := range firstBatch {
		firstBatch[index] = DueSchedule{ID: index + 1}
	}
	repository := &drainingSchedulerRepository{batches: [][]DueSchedule{firstBatch, {{ID: DueScheduleBatchSize + 1}}}}
	ctx, cancel := context.WithCancel(context.Background())
	waits := make(chan time.Duration, 1)
	controller := NewSchedulerController(repository, &dispatcherStub{}).WithRuntime(
		&sequenceClock{last: testTime()}, &cancelingWaiter{cancel: cancel, durations: waits}, zapLoggerAdapter{logger: zap.NewNop()},
	)

	controller.Start(ctx)
	select {
	case duration := <-waits:
		if duration != schedulerIdleInterval {
			t.Fatalf("idle wait = %s, want %s", duration, schedulerIdleInterval)
		}
	case <-time.After(time.Second):
		t.Fatal("controller did not reach idle after draining backlog")
	}
	select {
	case <-controller.Done():
	case <-time.After(time.Second):
		t.Fatal("controller did not exit after test cancellation")
	}
	listCalls, materialized := repository.snapshot()
	if listCalls != 3 || materialized != DueScheduleBatchSize+1 {
		t.Fatalf("controller backlog drain = listCalls %d materialized %d", listCalls, materialized)
	}
}

func TestSchedulerControllerWaitsAfterCommittedMaterializationThenPreAttemptError(t *testing.T) {
	wantErr := errors.New("materialization infrastructure failed")
	repository := &schedulerRepositoryStub{
		due:            []DueSchedule{{ID: 1}},
		materializeErr: wantErr,
		passCalls:      make(chan struct{}, 2),
	}
	ctx, cancel := context.WithCancel(context.Background())
	waits := make(chan time.Duration, 1)
	core, logs := observer.New(zap.DebugLevel)
	controller := NewSchedulerController(repository, &dispatcherStub{}).WithRuntime(
		&sequenceClock{last: testTime()}, &cancelingWaiter{cancel: cancel, durations: waits}, zapLoggerAdapter{logger: zap.New(core)},
	)

	controller.Start(ctx)
	select {
	case duration := <-waits:
		if duration != schedulerIdleInterval {
			t.Fatalf("error wait = %s, want %s", duration, schedulerIdleInterval)
		}
	case <-time.After(time.Second):
		t.Fatal("pre-attempt error retried instead of entering the normal wait")
	}
	<-controller.Done()
	if repository.materialized != 1 || logs.FilterMessage("Scheduled scan controller pass failed").Len() != 1 {
		t.Fatalf("controller error evidence = materialized %d logs %+v", repository.materialized, logs.All())
	}
	fields := logs.FilterMessage("Scheduled scan controller pass failed").All()[0].ContextMap()
	if fmt.Sprint(fields["scheduled_scan.materialized_count"]) != "1" || fields["scheduled_scan.attempted"] != false || fields["error_kind"] != "operation_failed" {
		t.Fatalf("controller error fields = %+v", fields)
	}
	if _, exposed := fields["error"]; exposed {
		t.Fatalf("controller log exposed a raw error: %+v", fields)
	}
	if logs.FilterLevelExact(zap.InfoLevel).Len() != 0 {
		t.Fatalf("controller progress/error pass emitted Info logs: %+v", logs.All())
	}
}

func TestSchedulerControllerCancellationStartsNoNewPass(t *testing.T) {
	repository := &schedulerRepositoryStub{passCalls: make(chan struct{}, 2)}
	waiter := &blockingWaiter{entered: make(chan time.Duration, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	controller := NewSchedulerController(repository, &dispatcherStub{}).WithRuntime(
		&sequenceClock{last: testTime()}, waiter, zapLoggerAdapter{logger: zap.NewNop()},
	)
	controller.Start(ctx)
	select {
	case <-waiter.entered:
	case <-time.After(time.Second):
		t.Fatal("controller did not enter idle wait")
	}
	cancel()
	select {
	case <-controller.Done():
	case <-time.After(time.Second):
		t.Fatal("controller did not stop after cancellation")
	}
	select {
	case <-repository.passCalls:
	default:
		t.Fatal("initial pass was not observed")
	}
	select {
	case <-repository.passCalls:
		t.Fatal("controller started a new pass after cancellation")
	default:
	}
}

func TestSchedulerAttemptUsesFixedFiveMinuteHandoffDeadline(t *testing.T) {
	now := testTime()
	targetID := 7
	repository := &schedulerRepositoryStub{
		candidate:  &OccurrenceCandidate{ID: 9, ScheduledScanID: 1, ScheduledFor: now},
		startInput: &FrozenDispatchInput{OccurrenceID: 9, ScheduledScanID: 1, ScheduledFor: now, ScanWorkflowID: "default", InputSource: scanapp.InputSourceScanSnapshot, TargetIDs: []int{targetID}, TargetScoped: true},
		recorded:   true,
	}
	dispatcher := &deadlineCapturingDispatcher{}
	controller := NewSchedulerController(repository, dispatcher).WithRuntime(
		&sequenceClock{times: []time.Time{now, now, now.Add(time.Second)}}, &cancelingWaiter{}, zapLoggerAdapter{logger: zap.NewNop()},
	)
	started := time.Now()

	result := controller.RunPass(context.Background())
	if result.Err != nil || !dispatcher.ok {
		t.Fatalf("RunPass() = %+v deadlinePresent=%t", result, dispatcher.ok)
	}
	remaining := dispatcher.deadline.Sub(started)
	if remaining < handoffDeadline-time.Second || remaining > handoffDeadline+time.Second {
		t.Fatalf("handoff deadline delta = %s, want %s", remaining, handoffDeadline)
	}
}

func TestSchedulerShutdownCancellationLeavesAttemptOutcomeUnknownWithoutReplay(t *testing.T) {
	now := testTime()
	targetID := 7
	repository := &schedulerRepositoryStub{
		candidate:      &OccurrenceCandidate{ID: 9, ScheduledScanID: 1, ScheduledFor: now},
		startInput:     &FrozenDispatchInput{OccurrenceID: 9, ScheduledScanID: 1, ScheduledFor: now, ScanWorkflowID: "default", InputSource: scanapp.InputSourceScanSnapshot, TargetIDs: []int{targetID}, TargetScoped: true},
		recorded:       true,
		honorRecordCtx: true,
	}
	dispatcher := &cancelBlockingDispatcher{started: make(chan struct{}, 1)}
	core, logs := observer.New(zap.DebugLevel)
	controller := NewSchedulerController(repository, dispatcher).WithRuntime(
		&sequenceClock{last: now}, &blockingWaiter{entered: make(chan time.Duration, 1)}, zapLoggerAdapter{logger: zap.New(core)},
	)
	ctx, cancel := context.WithCancel(context.Background())
	controller.Start(ctx)
	select {
	case <-dispatcher.started:
	case <-time.After(time.Second):
		t.Fatal("handoff did not start")
	}
	cancel()
	select {
	case <-controller.Done():
	case <-time.After(time.Second):
		t.Fatal("controller did not stop after shutdown cancellation")
	}
	if dispatcher.calls != 1 || repository.startCalls != 1 || len(repository.recordedOutcomes) != 0 {
		t.Fatalf("shutdown replay/outcome state = dispatches %d starts %d outcomes %+v", dispatcher.calls, repository.startCalls, repository.recordedOutcomes)
	}
	if logs.FilterMessage("Scheduled scan handoff did not complete").Len() != 0 || logs.FilterMessage("Scheduled scan controller pass failed").Len() != 1 {
		t.Fatalf("shutdown cancellation logs = %+v", logs.All())
	}
}

func TestSchedulerDoesNotWaitForCreatedScanLifecycleBeforeNextHandoff(t *testing.T) {
	now := testTime()
	targetID := 7
	repository := &queuedAttemptRepository{inputs: []FrozenDispatchInput{
		{OccurrenceID: 1, ScheduledScanID: 1, ScheduledFor: now, ScanWorkflowID: "default", InputSource: scanapp.InputSourceScanSnapshot, TargetIDs: []int{targetID}, TargetScoped: true},
		{OccurrenceID: 2, ScheduledScanID: 2, ScheduledFor: now.Add(time.Minute), ScanWorkflowID: "default", InputSource: scanapp.InputSourceScanSnapshot, TargetIDs: []int{targetID}, TargetScoped: true},
	}}
	dispatcher := &dispatcherStub{result: &scanapp.BatchScanResult{CreatedCount: 1, Scans: []scanapp.QueryScan{{ID: 99}}}}
	ctx, cancel := context.WithCancel(context.Background())
	waits := make(chan time.Duration, 1)
	controller := NewSchedulerController(repository, dispatcher).WithRuntime(
		&sequenceClock{last: now}, &cancelingWaiter{cancel: cancel, durations: waits}, zapLoggerAdapter{logger: zap.NewNop()},
	)
	controller.Start(ctx)
	select {
	case <-waits:
	case <-time.After(time.Second):
		t.Fatal("controller did not reach idle after two handoffs")
	}
	<-controller.Done()
	if dispatcher.calls != 2 || repository.recorded != 2 {
		t.Fatalf("independent handoffs = dispatches %d recorded %d", dispatcher.calls, repository.recorded)
	}
}
