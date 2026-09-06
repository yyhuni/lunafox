package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	systemdomain "github.com/yyhuni/lunafox/server/internal/modules/system/domain"
)

func TestServerLocationSchedulerMissingSuccessAndExactExpiryCycle(t *testing.T) {
	_, repository, lookup, runtime, cancel := newServerLocationSchedulerTest(t, nil)
	defer cancel()
	firstCall := lookup.waitCall(t)
	if runtime.TimerCount() != 0 {
		t.Fatalf("missing snapshot created a timer before immediate lookup: %d", runtime.TimerCount())
	}
	radius := 12.5
	firstCall.complete(ServerLocationLookupOutcome{ProviderAttempted: true, CompletedAt: runtime.NowUTC(), Location: &ServerLocationResolution{
		ObservedEgressIP: "8.8.8.8",
		Latitude:         37.4219999,
		Longitude:        -122.0840575,
		AccuracyRadiusKM: &radius,
		ProviderKey:      "freeipapi",
		ResolvedAt:       runtime.NowUTC(),
	}})
	replacement := repository.waitReplacement(t)
	if replacement.ObservedEgressIP != "8.8.8.8" || replacement.AccuracyRadiusKM == nil || *replacement.AccuracyRadiusKM != radius || replacement.ForcedExpired {
		t.Fatalf("persisted Server success = %#v", replacement)
	}
	runtime.waitForTimerCount(t, 1)
	if deadline := runtime.singleTimerDeadline(t); !deadline.Equal(runtime.NowUTC().Add(7 * 24 * time.Hour)) {
		t.Fatalf("success expiry timer = %s", deadline)
	}
	runtime.Advance(7*24*time.Hour - time.Nanosecond)
	lookup.assertNoCall(t)
	runtime.Advance(time.Nanosecond)
	secondCall := lookup.waitCall(t)
	if secondCall == firstCall {
		t.Fatal("expiry did not create a new logical self lookup")
	}
}

func TestServerLocationSchedulerRestoresCurrentSnapshotWithoutStartupLookup(t *testing.T) {
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	snapshot := testServerLocationSnapshot(base.Add(-24 * time.Hour))
	_, _, lookup, runtime, cancel := newServerLocationSchedulerTest(t, &snapshot)
	defer cancel()
	runtime.waitForTimerCount(t, 1)
	lookup.assertNoCall(t)
	want := snapshot.ResolvedAt.Add(7 * 24 * time.Hour)
	if deadline := runtime.singleTimerDeadline(t); !deadline.Equal(want) {
		t.Fatalf("restored snapshot timer = %s, want %s", deadline, want)
	}
}

func TestServerLocationSchedulerTreatsExactExpiryAsImmediatelyEligible(t *testing.T) {
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	snapshot := testServerLocationSnapshot(base.Add(-7 * 24 * time.Hour))
	_, _, lookup, runtime, cancel := newServerLocationSchedulerTest(t, &snapshot)
	defer cancel()
	lookup.waitCall(t)
	if runtime.TimerCount() != 0 {
		t.Fatalf("exactly expired snapshot delayed initial lookup with %d timers", runtime.TimerCount())
	}
}

func TestServerLocationSchedulerFailureRetainsSuccessAndWaitsForLaterCooldown(t *testing.T) {
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	snapshot := testServerLocationSnapshot(base.Add(-7 * 24 * time.Hour))
	scheduler, repository, lookup, runtime, cancel := newServerLocationSchedulerTest(t, &snapshot)
	defer cancel()
	_ = scheduler
	lookup.setNextAllowedAt(base.Add(10 * time.Minute))
	call := lookup.waitCall(t)
	call.complete(ServerLocationLookupOutcome{
		FailureClass:      "rate_limited",
		ProviderAttempted: true,
		CompletedAt:       base,
	})
	runtime.waitForTimerCount(t, 1)
	if deadline := runtime.singleTimerDeadline(t); !deadline.Equal(base.Add(10 * time.Minute)) {
		t.Fatalf("failure retry deadline = %s, want cooldown %s", deadline, base.Add(10*time.Minute))
	}
	if repository.replacementCount() != 0 || repository.markExpiredCount() != 0 {
		t.Fatalf("failed refresh mutated last success: replacements=%d marks=%d", repository.replacementCount(), repository.markExpiredCount())
	}
	stored := repository.currentSnapshot()
	if stored == nil || stored.ObservedEgressIP != snapshot.ObservedEgressIP || !stored.ResolvedAt.Equal(snapshot.ResolvedAt) {
		t.Fatalf("failed refresh changed stored success: %#v", stored)
	}
	runtime.Advance(5 * time.Minute)
	lookup.assertNoCall(t)
	runtime.Advance(5 * time.Minute)
	lookup.waitCall(t)
}

func TestServerLocationSchedulerFailureWithoutSuccessRemainsUnknown(t *testing.T) {
	_, repository, lookup, runtime, cancel := newServerLocationSchedulerTest(t, nil)
	defer cancel()
	call := lookup.waitCall(t)
	failedAt := runtime.NowUTC()
	call.complete(ServerLocationLookupOutcome{
		FailureClass:      "network",
		ProviderAttempted: true,
		CompletedAt:       failedAt,
	})

	runtime.waitForTimerCount(t, 1)
	if snapshot := repository.currentSnapshot(); snapshot != nil {
		t.Fatalf("failed first lookup persisted a placeholder: %#v", snapshot)
	}
	if repository.replacementCount() != 0 || repository.markExpiredCount() != 0 {
		t.Fatalf("failed first lookup mutated persistence: replacements=%d marks=%d", repository.replacementCount(), repository.markExpiredCount())
	}
	if deadline := runtime.singleTimerDeadline(t); !deadline.Equal(failedAt.Add(ServerLocationFailureRetryDelay)) {
		t.Fatalf("failed first lookup retry = %s, want %s", deadline, failedAt.Add(ServerLocationFailureRetryDelay))
	}
}

func TestServerLocationSchedulerQueueRejectionRetriesAfterFiveMinutes(t *testing.T) {
	_, _, lookup, runtime, cancel := newServerLocationSchedulerTestWithLookupErrors(t, nil, []error{errors.New("queue full")})
	defer cancel()
	runtime.waitForTimerCount(t, 1)
	lookup.assertNoCall(t)
	if deadline := runtime.singleTimerDeadline(t); !deadline.Equal(runtime.NowUTC().Add(5 * time.Minute)) {
		t.Fatalf("queue rejection retry = %s", deadline)
	}
	runtime.Advance(5 * time.Minute)
	lookup.waitCall(t)
}

func TestServerLocationSchedulerAllowsOnlyOnePendingOrInflightLookup(t *testing.T) {
	_, repository, lookup, runtime, cancel := newServerLocationSchedulerTest(t, nil)
	defer cancel()
	call := lookup.waitCall(t)
	runtime.Advance(30 * 24 * time.Hour)
	lookup.assertNoCall(t)
	if runtime.TimerCount() != 0 || lookup.activeCallCount() != 1 {
		t.Fatalf("in-flight accounting: timers=%d active=%d", runtime.TimerCount(), lookup.activeCallCount())
	}
	call.complete(ServerLocationLookupOutcome{ProviderAttempted: true, CompletedAt: runtime.NowUTC(), Location: &ServerLocationResolution{
		ObservedEgressIP: "1.1.1.1",
		Latitude:         1,
		Longitude:        2,
		ProviderKey:      "freeipapi",
		ResolvedAt:       runtime.NowUTC(),
	}})
	repository.waitReplacement(t)
	runtime.waitForTimerCount(t, 1)
}

func TestServerLocationSchedulerRetriesSnapshotRestoreWithoutProviderWork(t *testing.T) {
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	snapshot := testServerLocationSnapshot(base)
	repository := newSchedulerLocationRepository(&snapshot)
	repository.getErrors = []error{errors.New("database unavailable"), nil}
	lookup := newSchedulerLocationLookup(nil)
	runtime := newManualServerLocationRuntime(base)
	scheduler := mustNewServerLocationScheduler(t, repository, lookup, runtime)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { stopServerLocationScheduler(t, scheduler, runtime, cancel) })
	scheduler.Start(ctx)

	runtime.waitForTimerCount(t, 1)
	lookup.assertNoCall(t)
	runtime.Advance(5 * time.Minute)
	runtime.waitForTimerCount(t, 1)
	lookup.assertNoCall(t)
	if repository.getCallCount() != 2 {
		t.Fatalf("snapshot restore calls = %d, want 2", repository.getCallCount())
	}
}

func TestServerLocationSchedulerShutdownCleansTimerAndInflightLookup(t *testing.T) {
	t.Run("timer", func(t *testing.T) {
		base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
		snapshot := testServerLocationSnapshot(base)
		scheduler, _, lookup, runtime, cancel := newServerLocationSchedulerTest(t, &snapshot)
		runtime.waitForTimerCount(t, 1)
		cancel()
		waitServerLocationSchedulerDone(t, scheduler)
		if runtime.TimerCount() != 0 {
			t.Fatalf("timer count after shutdown = %d", runtime.TimerCount())
		}
		lookup.assertNoCall(t)
	})

	t.Run("inflight", func(t *testing.T) {
		scheduler, _, lookup, runtime, cancel := newServerLocationSchedulerTest(t, nil)
		call := lookup.waitCall(t)
		cancel()
		waitServerLocationSchedulerDone(t, scheduler)
		select {
		case <-call.context.Done():
		case <-time.After(time.Second):
			t.Fatal("in-flight self lookup context was not canceled")
		}
		if runtime.TimerCount() != 0 {
			t.Fatalf("timer count after in-flight shutdown = %d", runtime.TimerCount())
		}
	})
}

func newServerLocationSchedulerTest(
	t *testing.T,
	snapshot *systemdomain.ServerLocationSnapshot,
) (*ServerLocationScheduler, *schedulerLocationRepository, *schedulerLocationLookup, *manualServerLocationRuntime, context.CancelFunc) {
	t.Helper()
	return newServerLocationSchedulerTestWithLookupErrors(t, snapshot, nil)
}

func newServerLocationSchedulerTestWithLookupErrors(
	t *testing.T,
	snapshot *systemdomain.ServerLocationSnapshot,
	submitErrors []error,
) (*ServerLocationScheduler, *schedulerLocationRepository, *schedulerLocationLookup, *manualServerLocationRuntime, context.CancelFunc) {
	t.Helper()
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	repository := newSchedulerLocationRepository(snapshot)
	lookup := newSchedulerLocationLookup(submitErrors)
	runtime := newManualServerLocationRuntime(base)
	scheduler := mustNewServerLocationScheduler(t, repository, lookup, runtime)
	ctx, cancel := context.WithCancel(context.Background())
	scheduler.Start(ctx)
	t.Cleanup(func() { stopServerLocationScheduler(t, scheduler, runtime, cancel) })
	return scheduler, repository, lookup, runtime, cancel
}

func mustNewServerLocationScheduler(
	t *testing.T,
	repository systemdomain.ServerLocationRepository,
	lookup ServerLocationLookup,
	runtime ServerLocationSchedulerRuntime,
) *ServerLocationScheduler {
	t.Helper()
	scheduler, err := NewServerLocationScheduler(repository, lookup, runtime)
	if err != nil {
		t.Fatalf("NewServerLocationScheduler: %v", err)
	}
	return scheduler
}

func stopServerLocationScheduler(t *testing.T, scheduler *ServerLocationScheduler, runtime *manualServerLocationRuntime, cancel context.CancelFunc) {
	t.Helper()
	cancel()
	waitServerLocationSchedulerDone(t, scheduler)
	if runtime.TimerCount() != 0 {
		t.Errorf("live Server location timers after cleanup = %d", runtime.TimerCount())
	}
}

func waitServerLocationSchedulerDone(t *testing.T, scheduler *ServerLocationScheduler) {
	t.Helper()
	select {
	case <-scheduler.Done():
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Server location scheduler shutdown")
	}
}

func testServerLocationSnapshot(resolvedAt time.Time) systemdomain.ServerLocationSnapshot {
	return systemdomain.ServerLocationSnapshot{
		ObservedEgressIP: "8.8.8.8",
		Latitude:         1,
		Longitude:        2,
		ProviderKey:      "freeipapi",
		ResolvedAt:       resolvedAt,
		UpdatedAt:        resolvedAt,
	}
}

type schedulerLocationRepository struct {
	mu           sync.Mutex
	snapshot     *systemdomain.ServerLocationSnapshot
	getErrors    []error
	getCalls     int
	replacements chan systemdomain.ServerLocationSnapshot
	markCalls    int
}

func newSchedulerLocationRepository(snapshot *systemdomain.ServerLocationSnapshot) *schedulerLocationRepository {
	return &schedulerLocationRepository{
		snapshot:     cloneServerLocationSnapshot(snapshot),
		replacements: make(chan systemdomain.ServerLocationSnapshot, 8),
	}
}

func (repository *schedulerLocationRepository) Get(context.Context) (*systemdomain.ServerLocationSnapshot, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.getCalls++
	if len(repository.getErrors) > 0 {
		err := repository.getErrors[0]
		repository.getErrors = repository.getErrors[1:]
		if err != nil {
			return nil, err
		}
	}
	return cloneServerLocationSnapshot(repository.snapshot), nil
}

func (repository *schedulerLocationRepository) Replace(_ context.Context, snapshot systemdomain.ServerLocationSnapshot) error {
	repository.mu.Lock()
	repository.snapshot = cloneServerLocationSnapshot(&snapshot)
	repository.mu.Unlock()
	repository.replacements <- snapshot
	return nil
}

func (repository *schedulerLocationRepository) MarkExpired(context.Context) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.markCalls++
	if repository.snapshot != nil {
		repository.snapshot.ForcedExpired = true
	}
	return nil
}

func (repository *schedulerLocationRepository) waitReplacement(t *testing.T) systemdomain.ServerLocationSnapshot {
	t.Helper()
	select {
	case snapshot := <-repository.replacements:
		return snapshot
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Server location replacement")
		return systemdomain.ServerLocationSnapshot{}
	}
}

func (repository *schedulerLocationRepository) replacementCount() int {
	return len(repository.replacements)
}

func (repository *schedulerLocationRepository) markExpiredCount() int {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return repository.markCalls
}

func (repository *schedulerLocationRepository) currentSnapshot() *systemdomain.ServerLocationSnapshot {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return cloneServerLocationSnapshot(repository.snapshot)
}

func (repository *schedulerLocationRepository) getCallCount() int {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return repository.getCalls
}

func cloneServerLocationSnapshot(snapshot *systemdomain.ServerLocationSnapshot) *systemdomain.ServerLocationSnapshot {
	if snapshot == nil {
		return nil
	}
	copy := *snapshot
	if snapshot.AccuracyRadiusKM != nil {
		radius := *snapshot.AccuracyRadiusKM
		copy.AccuracyRadiusKM = &radius
	}
	return &copy
}

type schedulerLocationLookup struct {
	mu           sync.Mutex
	submitErrors []error
	nextAllowed  time.Time
	calls        chan *schedulerLocationLookupCall
	active       int
}

type schedulerLocationLookupCall struct {
	context    context.Context
	completion func(ServerLocationLookupOutcome)
	once       sync.Once
}

func newSchedulerLocationLookup(submitErrors []error) *schedulerLocationLookup {
	return &schedulerLocationLookup{
		submitErrors: append([]error(nil), submitErrors...),
		calls:        make(chan *schedulerLocationLookupCall, 16),
	}
}

func (lookup *schedulerLocationLookup) SubmitSelf(ctx context.Context, completion func(ServerLocationLookupOutcome)) error {
	lookup.mu.Lock()
	if len(lookup.submitErrors) > 0 {
		err := lookup.submitErrors[0]
		lookup.submitErrors = lookup.submitErrors[1:]
		if err != nil {
			lookup.mu.Unlock()
			return err
		}
	}
	call := &schedulerLocationLookupCall{context: ctx}
	call.completion = func(outcome ServerLocationLookupOutcome) {
		call.once.Do(func() {
			lookup.mu.Lock()
			lookup.active--
			lookup.mu.Unlock()
			completion(outcome)
		})
	}
	lookup.active++
	lookup.mu.Unlock()
	lookup.calls <- call
	go func() {
		<-ctx.Done()
		call.complete(ServerLocationLookupOutcome{Canceled: true, CompletedAt: time.Now().UTC()})
	}()
	return nil
}

func (lookup *schedulerLocationLookup) NextAllowedAt() time.Time {
	lookup.mu.Lock()
	defer lookup.mu.Unlock()
	return lookup.nextAllowed
}

func (lookup *schedulerLocationLookup) setNextAllowedAt(instant time.Time) {
	lookup.mu.Lock()
	lookup.nextAllowed = instant
	lookup.mu.Unlock()
}

func (lookup *schedulerLocationLookup) waitCall(t *testing.T) *schedulerLocationLookupCall {
	t.Helper()
	select {
	case call := <-lookup.calls:
		return call
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Server self lookup")
		return nil
	}
}

func (lookup *schedulerLocationLookup) assertNoCall(t *testing.T) {
	t.Helper()
	select {
	case <-lookup.calls:
		t.Fatal("unexpected Server self lookup")
	case <-time.After(10 * time.Millisecond):
	}
}

func (lookup *schedulerLocationLookup) activeCallCount() int {
	lookup.mu.Lock()
	defer lookup.mu.Unlock()
	return lookup.active
}

func (call *schedulerLocationLookupCall) complete(outcome ServerLocationLookupOutcome) {
	call.completion(outcome)
}

type manualServerLocationRuntime struct {
	mu     sync.Mutex
	now    time.Time
	timers map[*manualServerLocationTimer]struct{}
}

type manualServerLocationTimer struct {
	runtime  *manualServerLocationRuntime
	deadline time.Time
	channel  chan time.Time
	stopped  bool
	fired    bool
}

func newManualServerLocationRuntime(now time.Time) *manualServerLocationRuntime {
	return &manualServerLocationRuntime{now: now.UTC(), timers: make(map[*manualServerLocationTimer]struct{})}
}

func (runtime *manualServerLocationRuntime) NowUTC() time.Time {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return runtime.now
}

func (runtime *manualServerLocationRuntime) NewTimer(delay time.Duration) ServerLocationTimer {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	timer := &manualServerLocationTimer{
		runtime:  runtime,
		deadline: runtime.now.Add(delay),
		channel:  make(chan time.Time, 1),
	}
	runtime.timers[timer] = struct{}{}
	return timer
}

func (runtime *manualServerLocationRuntime) Advance(duration time.Duration) {
	runtime.mu.Lock()
	runtime.now = runtime.now.Add(duration)
	now := runtime.now
	due := make([]*manualServerLocationTimer, 0)
	for timer := range runtime.timers {
		if !timer.stopped && !timer.fired && !timer.deadline.After(now) {
			timer.fired = true
			delete(runtime.timers, timer)
			due = append(due, timer)
		}
	}
	runtime.mu.Unlock()
	for _, timer := range due {
		timer.channel <- now
	}
}

func (runtime *manualServerLocationRuntime) TimerCount() int {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return len(runtime.timers)
}

func (runtime *manualServerLocationRuntime) waitForTimerCount(t *testing.T, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for runtime.TimerCount() != count {
		if time.Now().After(deadline) {
			t.Fatalf("Server location timer count = %d, want %d", runtime.TimerCount(), count)
		}
		time.Sleep(time.Millisecond)
	}
}

func (runtime *manualServerLocationRuntime) singleTimerDeadline(t *testing.T) time.Time {
	t.Helper()
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if len(runtime.timers) != 1 {
		t.Fatalf("timer count = %d, want 1", len(runtime.timers))
	}
	for timer := range runtime.timers {
		return timer.deadline
	}
	return time.Time{}
}

func (timer *manualServerLocationTimer) Channel() <-chan time.Time { return timer.channel }

func (timer *manualServerLocationTimer) Stop() bool {
	timer.runtime.mu.Lock()
	defer timer.runtime.mu.Unlock()
	if timer.stopped || timer.fired {
		return false
	}
	timer.stopped = true
	delete(timer.runtime.timers, timer)
	return true
}
