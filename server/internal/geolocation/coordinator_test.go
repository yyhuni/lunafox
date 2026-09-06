package geolocation

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCoordinatorDefaultsMatchProcessLocalLimits(t *testing.T) {
	options := defaultCoordinatorOptions()
	if options.queueCapacity != 128 || options.maxInFlight != 4 || options.maxAttempts != 60 || options.attemptWindow != time.Minute || options.cacheCapacity != 1024 || options.successTTL != 7*24*time.Hour || options.negativeTTL != 5*time.Minute {
		t.Fatalf("unexpected coordinator defaults: %#v", options)
	}
}

func TestCoordinatorDefaultWaitingDequeHoldsExactly128Entries(t *testing.T) {
	options := defaultCoordinatorOptions()
	options.maxInFlight = 1
	coordinator, provider, _ := newTestCoordinator(t, options)
	if err := coordinator.SubmitSelf(context.Background(), func(LookupOutcome) {}); err != nil {
		t.Fatalf("SubmitSelf active: %v", err)
	}
	provider.nextCall(t)
	for index := range 128 {
		if err := coordinator.SubmitSelf(context.Background(), func(LookupOutcome) {}); err != nil {
			t.Fatalf("SubmitSelf waiting %d: %v", index+1, err)
		}
	}
	assertCoordinatorSnapshot(t, coordinator, func(snapshot coordinatorSnapshot) bool {
		return snapshot.Active == 1 && snapshot.Queued == 128
	})
	if err := coordinator.SubmitSelf(context.Background(), func(LookupOutcome) {}); !errors.Is(err, ErrCoordinatorQueueFull) {
		t.Fatalf("129th waiting submission = %v, want %v", err, ErrCoordinatorQueueFull)
	}
}

func TestCoordinatorCountsWaitingSeparatelyAndDropsNewestWhenFIFOIsFull(t *testing.T) {
	coordinator, provider, _ := newTestCoordinator(t, coordinatorOptions{
		queueCapacity: 2,
		maxInFlight:   1,
		maxAttempts:   10,
		attemptWindow: time.Minute,
		cacheCapacity: 8,
		successTTL:    7 * 24 * time.Hour,
		negativeTTL:   5 * time.Minute,
	})

	first := submitTestIP(t, coordinator, "8.8.8.8")
	firstCall := provider.nextCall(t)
	assertCoordinatorSnapshot(t, coordinator, func(snapshot coordinatorSnapshot) bool {
		return snapshot.Active == 1 && snapshot.Queued == 0
	})
	second := submitTestIP(t, coordinator, "1.1.1.1")
	third := submitTestIP(t, coordinator, "9.9.9.9")
	assertCoordinatorSnapshot(t, coordinator, func(snapshot coordinatorSnapshot) bool {
		return snapshot.Active == 1 && snapshot.Queued == 2 && snapshot.Flights == 3
	})

	err := coordinator.SubmitIP(context.Background(), "208.67.222.222", func(LookupOutcome) {
		t.Error("rejected submission must not receive a completion")
	})
	if !errors.Is(err, ErrCoordinatorQueueFull) {
		t.Fatalf("full queue submission error = %v, want %v", err, ErrCoordinatorQueueFull)
	}

	firstCall.complete(testLocation(firstCall.targetIP, firstCall.startedAt), nil)
	requireSuccessOutcome(t, first, false)
	secondCall := provider.nextCall(t)
	if secondCall.targetIP != "1.1.1.1" {
		t.Fatalf("second provider target = %q, want FIFO target 1.1.1.1", secondCall.targetIP)
	}
	secondCall.complete(testLocation(secondCall.targetIP, secondCall.startedAt), nil)
	requireSuccessOutcome(t, second, false)
	thirdCall := provider.nextCall(t)
	if thirdCall.targetIP != "9.9.9.9" {
		t.Fatalf("third provider target = %q, want FIFO target 9.9.9.9", thirdCall.targetIP)
	}
	thirdCall.complete(testLocation(thirdCall.targetIP, thirdCall.startedAt), nil)
	requireSuccessOutcome(t, third, false)
}

func TestCoordinatorCanceledQueuedFlightReleasesAdmissionWithoutWaitingForAPermit(t *testing.T) {
	options := testCoordinatorOptions()
	options.maxInFlight = 1
	options.queueCapacity = 1
	coordinator, provider, _ := newTestCoordinator(t, options)
	active := submitTestIP(t, coordinator, "8.8.8.8")
	activeCall := provider.nextCall(t)

	canceledContext, cancel := context.WithCancel(context.Background())
	canceledCompletion := make(chan LookupOutcome, 1)
	if err := coordinator.SubmitIP(canceledContext, "1.1.1.1", func(outcome LookupOutcome) { canceledCompletion <- outcome }); err != nil {
		t.Fatalf("SubmitIP canceled waiter: %v", err)
	}
	cancel()
	assertCoordinatorSnapshot(t, coordinator, func(snapshot coordinatorSnapshot) bool {
		return snapshot.Active == 1 && snapshot.Queued == 0 && snapshot.Flights == 1
	})

	replacement := submitTestIP(t, coordinator, "9.9.9.9")
	activeCall.complete(testLocation(activeCall.targetIP, activeCall.startedAt), nil)
	requireSuccessOutcome(t, active, false)
	replacementCall := provider.nextCall(t)
	if replacementCall.targetIP != "9.9.9.9" {
		t.Fatalf("replacement target = %q, want 9.9.9.9", replacementCall.targetIP)
	}
	replacementCall.complete(testLocation(replacementCall.targetIP, replacementCall.startedAt), nil)
	requireSuccessOutcome(t, replacement, false)
	canceledOutcome := receiveOutcome(t, canceledCompletion)
	if !canceledOutcome.Canceled || canceledOutcome.ProviderAttempted || canceledOutcome.Location != nil || canceledOutcome.Failure != nil {
		t.Fatalf("canceled queued waiter outcome = %#v", canceledOutcome)
	}
}

func TestCoordinatorExistingFlightCanBeJoinedWhileQueueIsFull(t *testing.T) {
	options := testCoordinatorOptions()
	options.maxInFlight = 1
	options.queueCapacity = 1
	coordinator, provider, _ := newTestCoordinator(t, options)
	active := submitTestIP(t, coordinator, "8.8.8.8")
	activeCall := provider.nextCall(t)
	firstWaiter := submitTestIP(t, coordinator, "1.1.1.1")
	secondWaiter := submitTestIP(t, coordinator, "1.1.1.1")
	if err := coordinator.SubmitIP(context.Background(), "9.9.9.9", func(LookupOutcome) {}); !errors.Is(err, ErrCoordinatorQueueFull) {
		t.Fatalf("new flight with full queue = %v, want %v", err, ErrCoordinatorQueueFull)
	}

	activeCall.complete(testLocation(activeCall.targetIP, activeCall.startedAt), nil)
	requireSuccessOutcome(t, active, false)
	sharedCall := provider.nextCall(t)
	if sharedCall.targetIP != "1.1.1.1" {
		t.Fatalf("shared target = %q, want 1.1.1.1", sharedCall.targetIP)
	}
	sharedCall.complete(testLocation(sharedCall.targetIP, sharedCall.startedAt), nil)
	requireSuccessOutcome(t, firstWaiter, false)
	requireSuccessOutcome(t, secondWaiter, false)
	provider.assertNoCall(t)
}

func TestCoordinatorRunsAtMostFourProviderAttempts(t *testing.T) {
	coordinator, provider, _ := newTestCoordinator(t, testCoordinatorOptions())
	outcomes := make(map[string]<-chan LookupOutcome, 5)
	for _, ip := range []string{"8.8.8.8", "1.1.1.1", "9.9.9.9", "208.67.222.222", "4.2.2.2"} {
		outcomes[ip] = submitTestIP(t, coordinator, ip)
	}

	calls := make([]*controlledLookupCall, 0, 4)
	for range 4 {
		calls = append(calls, provider.nextCall(t))
	}
	assertCoordinatorSnapshot(t, coordinator, func(snapshot coordinatorSnapshot) bool {
		return snapshot.Active == 4 && snapshot.Queued == 1
	})
	provider.assertNoCall(t)

	calls[0].complete(testLocation(calls[0].targetIP, calls[0].startedAt), nil)
	requireSuccessOutcome(t, outcomes[calls[0].targetIP], false)
	fifthCall := provider.nextCall(t)
	if provider.peak.Load() > 4 {
		t.Fatalf("peak provider concurrency = %d, want at most 4", provider.peak.Load())
	}

	for _, call := range calls[1:] {
		call.complete(testLocation(call.targetIP, call.startedAt), nil)
		requireSuccessOutcome(t, outcomes[call.targetIP], false)
	}
	fifthCall.complete(testLocation(fifthCall.targetIP, fifthCall.startedAt), nil)
	requireSuccessOutcome(t, outcomes[fifthCall.targetIP], false)
}

func TestCoordinatorRollingAttemptGateUsesExactWindowBoundary(t *testing.T) {
	options := testCoordinatorOptions()
	options.maxAttempts = 2
	options.maxInFlight = 2
	coordinator, provider, runtime := newTestCoordinator(t, options)
	first := submitTestIP(t, coordinator, "8.8.8.8")
	second := submitTestIP(t, coordinator, "1.1.1.1")
	third := submitTestIP(t, coordinator, "9.9.9.9")

	firstCall := provider.nextCall(t)
	secondCall := provider.nextCall(t)
	firstCall.complete(testLocation(firstCall.targetIP, firstCall.startedAt), nil)
	secondCall.complete(testLocation(secondCall.targetIP, secondCall.startedAt), nil)
	requireSuccessOutcome(t, first, false)
	requireSuccessOutcome(t, second, false)
	runtime.waitForTimerCount(t, 1)
	provider.assertNoCall(t)

	runtime.Advance(time.Minute - time.Nanosecond)
	provider.assertNoCall(t)
	runtime.Advance(time.Nanosecond)
	thirdCall := provider.nextCall(t)
	thirdCall.complete(testLocation(thirdCall.targetIP, runtime.Now()), nil)
	requireSuccessOutcome(t, third, false)
}

func TestCoordinatorRateLimitCooldownOnlyExtendsAndDoesNotCancelActiveCalls(t *testing.T) {
	options := testCoordinatorOptions()
	options.maxInFlight = 2
	coordinator, provider, runtime := newTestCoordinator(t, options)
	outcomes := map[string]<-chan LookupOutcome{
		"8.8.8.8": submitTestIP(t, coordinator, "8.8.8.8"),
		"1.1.1.1": submitTestIP(t, coordinator, "1.1.1.1"),
	}
	firstCall := provider.nextCall(t)
	secondCall := provider.nextCall(t)

	longerBoundary := runtime.Now().Add(2 * time.Minute)
	firstCall.complete(Location{}, &LookupError{Class: FailureRateLimited, RetryAfterAt: longerBoundary})
	requireFailureOutcome(t, outcomes[firstCall.targetIP], FailureRateLimited, false)
	if secondCall.context.Err() != nil {
		t.Fatalf("429 canceled an already active call: %v", secondCall.context.Err())
	}
	secondCall.complete(Location{}, &LookupError{Class: FailureRateLimited, RetryAfterAt: runtime.Now().Add(time.Minute)})
	requireFailureOutcome(t, outcomes[secondCall.targetIP], FailureRateLimited, false)
	if got := coordinator.NextAllowedAt(); !got.Equal(longerBoundary) {
		t.Fatalf("cooldown boundary = %s, want monotonic %s", got, longerBoundary)
	}

	third := submitTestIP(t, coordinator, "9.9.9.9")
	runtime.waitForTimerCount(t, 1)
	runtime.Advance(time.Minute)
	provider.assertNoCall(t)
	if got := coordinator.NextAllowedAt(); !got.Equal(longerBoundary) {
		t.Fatalf("cooldown shortened after one minute: %s", got)
	}
	runtime.Advance(time.Minute)
	thirdCall := provider.nextCall(t)
	thirdCall.complete(testLocation(thirdCall.targetIP, runtime.Now()), nil)
	requireSuccessOutcome(t, third, false)
}

func TestCoordinatorSingleFlightFansOutAndThenUsesSuccessCache(t *testing.T) {
	coordinator, provider, runtime := newTestCoordinator(t, testCoordinatorOptions())
	first := submitTestIP(t, coordinator, "8.8.8.8")
	second := submitTestIP(t, coordinator, "::ffff:8.8.8.8")
	call := provider.nextCall(t)
	provider.assertNoCall(t)
	assertCoordinatorSnapshot(t, coordinator, func(snapshot coordinatorSnapshot) bool {
		return snapshot.Active == 1 && snapshot.Flights == 1
	})

	call.complete(testLocation("8.8.8.8", runtime.Now()), nil)
	requireSuccessOutcome(t, first, false)
	requireSuccessOutcome(t, second, false)
	third := submitTestIP(t, coordinator, "8.8.8.8")
	requireSuccessOutcome(t, third, true)
	provider.assertNoCall(t)
}

func TestCoordinatorEmitsOneBoundedDiagnosticPerActualProviderAttempt(t *testing.T) {
	options := testCoordinatorOptions()
	events := make(chan providerAttemptEvent, 4)
	options.attemptObserver = func(event providerAttemptEvent) { events <- event }
	coordinator, provider, runtime := newTestCoordinator(t, options)
	first := submitTestIP(t, coordinator, "8.8.8.8")
	joined := submitTestIP(t, coordinator, "8.8.8.8")
	call := provider.nextCall(t)
	call.complete(testLocation(call.targetIP, runtime.Now()), nil)
	requireSuccessOutcome(t, first, false)
	requireSuccessOutcome(t, joined, false)

	select {
	case event := <-events:
		if event.ProviderKey != FreeIPAPIProviderKey || event.ObservedIP != "8.8.8.8" || event.Outcome != "success" || event.FailureClass != "" || !event.AttemptedAt.Equal(call.startedAt) {
			t.Fatalf("success diagnostic = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for success diagnostic")
	}
	cached := submitTestIP(t, coordinator, "8.8.8.8")
	requireSuccessOutcome(t, cached, true)
	select {
	case event := <-events:
		t.Fatalf("cache hit emitted provider diagnostic: %#v", event)
	case <-time.After(10 * time.Millisecond):
	}

	failed := submitTestIP(t, coordinator, "1.1.1.1")
	failedCall := provider.nextCall(t)
	failedCall.complete(Location{}, errors.New("raw provider detail must not enter the diagnostic"))
	requireFailureOutcome(t, failed, FailureNetwork, false)
	select {
	case event := <-events:
		if event.ProviderKey != FreeIPAPIProviderKey || event.ObservedIP != "1.1.1.1" || event.Outcome != "failed" || event.FailureClass != FailureNetwork || !event.AttemptedAt.Equal(failedCall.startedAt) {
			t.Fatalf("failure diagnostic = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for failure diagnostic")
	}
}

func TestCoordinatorCachesProviderNegativesButNotAdmissionOrCancellation(t *testing.T) {
	options := testCoordinatorOptions()
	options.maxInFlight = 1
	options.queueCapacity = 1
	coordinator, provider, runtime := newTestCoordinator(t, options)
	first := submitTestIP(t, coordinator, "8.8.8.8")
	call := provider.nextCall(t)
	call.complete(Location{}, &LookupError{Class: FailureNetwork})
	requireFailureOutcome(t, first, FailureNetwork, false)

	cached := submitTestIP(t, coordinator, "8.8.8.8")
	requireFailureOutcome(t, cached, FailureNetwork, true)
	provider.assertNoCall(t)
	runtime.Advance(5 * time.Minute)
	retry := submitTestIP(t, coordinator, "8.8.8.8")
	retryCall := provider.nextCall(t)
	retryCall.complete(testLocation(retryCall.targetIP, runtime.Now()), nil)
	requireSuccessOutcome(t, retry, false)

	active := submitTestIP(t, coordinator, "1.1.1.1")
	activeCall := provider.nextCall(t)
	queued := submitTestIP(t, coordinator, "9.9.9.9")
	err := coordinator.SubmitIP(context.Background(), "208.67.222.222", func(LookupOutcome) {})
	if !errors.Is(err, ErrCoordinatorQueueFull) {
		t.Fatalf("queue rejection = %v, want %v", err, ErrCoordinatorQueueFull)
	}
	activeCall.complete(testLocation(activeCall.targetIP, runtime.Now()), nil)
	requireSuccessOutcome(t, active, false)
	queuedCall := provider.nextCall(t)
	queuedCall.complete(testLocation(queuedCall.targetIP, runtime.Now()), nil)
	requireSuccessOutcome(t, queued, false)
	if snapshot := coordinator.snapshot(); snapshot.CacheEntries != 3 {
		t.Fatalf("cache entries = %d, want only three provider results", snapshot.CacheEntries)
	}
}

func TestCoordinatorShutdownDropsQueuedWorkCancelsActiveAndCleansTimers(t *testing.T) {
	options := testCoordinatorOptions()
	options.maxInFlight = 1
	options.maxAttempts = 1
	coordinator, provider, runtime := newTestCoordinatorWithoutCleanup(t, options)
	active := submitTestIP(t, coordinator, "8.8.8.8")
	activeCall := provider.nextCall(t)
	queued := submitTestIP(t, coordinator, "1.1.1.1")
	assertCoordinatorSnapshot(t, coordinator, func(snapshot coordinatorSnapshot) bool {
		return snapshot.Active == 1 && snapshot.Queued == 1
	})

	shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- coordinator.Shutdown(shutdownContext) }()

	queuedOutcome := receiveOutcome(t, queued)
	if !queuedOutcome.Canceled || queuedOutcome.ProviderAttempted || queuedOutcome.FromCache {
		t.Fatalf("queued shutdown outcome = %#v", queuedOutcome)
	}
	activeOutcome := receiveOutcome(t, active)
	if activeOutcome.Failure == nil || activeOutcome.Failure.Class != FailureCanceled || !activeOutcome.Canceled || !activeOutcome.ProviderAttempted {
		t.Fatalf("active shutdown outcome = %#v", activeOutcome)
	}
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("Shutdown: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Shutdown did not finish after active provider cancellation")
	}
	select {
	case <-activeCall.returned:
	case <-time.After(time.Second):
		t.Fatal("active provider goroutine did not return")
	}
	if runtime.TimerCount() != 0 {
		t.Fatalf("live coordinator timers after shutdown = %d", runtime.TimerCount())
	}
	if err := coordinator.SubmitIP(context.Background(), "9.9.9.9", func(LookupOutcome) {}); !errors.Is(err, ErrCoordinatorClosed) {
		t.Fatalf("post-shutdown submission = %v, want %v", err, ErrCoordinatorClosed)
	}
}

func TestCoordinatorShutdownStopsARateGateTimer(t *testing.T) {
	options := testCoordinatorOptions()
	options.maxInFlight = 1
	options.maxAttempts = 1
	coordinator, provider, runtime := newTestCoordinatorWithoutCleanup(t, options)
	first := submitTestIP(t, coordinator, "8.8.8.8")
	firstCall := provider.nextCall(t)
	second := submitTestIP(t, coordinator, "1.1.1.1")
	firstCall.complete(testLocation(firstCall.targetIP, firstCall.startedAt), nil)
	requireSuccessOutcome(t, first, false)
	runtime.waitForTimerCount(t, 1)

	shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := coordinator.Shutdown(shutdownContext); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	secondOutcome := receiveOutcome(t, second)
	if !secondOutcome.Canceled || secondOutcome.ProviderAttempted {
		t.Fatalf("rate-gated shutdown outcome = %#v", secondOutcome)
	}
	if runtime.TimerCount() != 0 {
		t.Fatalf("rate-gate timer leaked after shutdown: %d", runtime.TimerCount())
	}
}

func testCoordinatorOptions() coordinatorOptions {
	return coordinatorOptions{
		queueCapacity: 8,
		maxInFlight:   4,
		maxAttempts:   60,
		attemptWindow: time.Minute,
		cacheCapacity: 16,
		successTTL:    7 * 24 * time.Hour,
		negativeTTL:   5 * time.Minute,
	}
}

func newTestCoordinator(t *testing.T, options coordinatorOptions) (*Coordinator, *controlledLookupProvider, *manualCoordinatorRuntime) {
	t.Helper()
	coordinator, provider, runtime := newTestCoordinatorWithoutCleanup(t, options)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := coordinator.Shutdown(ctx); err != nil {
			t.Errorf("coordinator cleanup: %v", err)
		}
	})
	return coordinator, provider, runtime
}

func newTestCoordinatorWithoutCleanup(t *testing.T, options coordinatorOptions) (*Coordinator, *controlledLookupProvider, *manualCoordinatorRuntime) {
	t.Helper()
	runtime := newManualCoordinatorRuntime(time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC))
	provider := newControlledLookupProvider(runtime)
	coordinator, err := newCoordinator(provider, runtime, options)
	if err != nil {
		t.Fatalf("newCoordinator: %v", err)
	}
	return coordinator, provider, runtime
}

func submitTestIP(t *testing.T, coordinator *Coordinator, ip string) <-chan LookupOutcome {
	t.Helper()
	outcomes := make(chan LookupOutcome, 1)
	if err := coordinator.SubmitIP(context.Background(), ip, func(outcome LookupOutcome) { outcomes <- outcome }); err != nil {
		t.Fatalf("SubmitIP(%q): %v", ip, err)
	}
	return outcomes
}

func requireSuccessOutcome(t *testing.T, outcomes <-chan LookupOutcome, fromCache bool) LookupOutcome {
	t.Helper()
	outcome := receiveOutcome(t, outcomes)
	if outcome.Location == nil || outcome.Failure != nil || outcome.Canceled || outcome.FromCache != fromCache || outcome.ProviderAttempted == fromCache {
		t.Fatalf("success outcome = %#v, want fromCache=%t", outcome, fromCache)
	}
	return outcome
}

func requireFailureOutcome(t *testing.T, outcomes <-chan LookupOutcome, class FailureClass, fromCache bool) LookupOutcome {
	t.Helper()
	outcome := receiveOutcome(t, outcomes)
	if outcome.Location != nil || outcome.Failure == nil || outcome.Failure.Class != class || outcome.Canceled || outcome.FromCache != fromCache || outcome.ProviderAttempted == fromCache {
		t.Fatalf("failure outcome = %#v, want class=%s fromCache=%t", outcome, class, fromCache)
	}
	return outcome
}

func receiveOutcome(t *testing.T, outcomes <-chan LookupOutcome) LookupOutcome {
	t.Helper()
	select {
	case outcome := <-outcomes:
		return outcome
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for lookup outcome")
		return LookupOutcome{}
	}
}

func assertCoordinatorSnapshot(t *testing.T, coordinator *Coordinator, predicate func(coordinatorSnapshot) bool) coordinatorSnapshot {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		snapshot := coordinator.snapshot()
		if predicate(snapshot) {
			return snapshot
		}
		if time.Now().After(deadline) {
			t.Fatalf("coordinator snapshot did not reach expected state: %#v", snapshot)
		}
		time.Sleep(time.Millisecond)
	}
}

type controlledLookupProvider struct {
	runtime *manualCoordinatorRuntime
	calls   chan *controlledLookupCall
	active  atomic.Int32
	peak    atomic.Int32
}

type controlledLookupCall struct {
	targetIP  string
	startedAt time.Time
	context   context.Context
	result    chan controlledLookupResult
	returned  chan struct{}
}

type controlledLookupResult struct {
	location Location
	err      error
}

func newControlledLookupProvider(runtime *manualCoordinatorRuntime) *controlledLookupProvider {
	return &controlledLookupProvider{runtime: runtime, calls: make(chan *controlledLookupCall, 256)}
}

func (provider *controlledLookupProvider) Lookup(ctx context.Context, targetIP string) (Location, error) {
	active := provider.active.Add(1)
	for {
		peak := provider.peak.Load()
		if active <= peak || provider.peak.CompareAndSwap(peak, active) {
			break
		}
	}
	defer provider.active.Add(-1)
	call := &controlledLookupCall{
		targetIP:  targetIP,
		startedAt: provider.runtime.Now(),
		context:   ctx,
		result:    make(chan controlledLookupResult, 1),
		returned:  make(chan struct{}),
	}
	defer close(call.returned)
	select {
	case provider.calls <- call:
	case <-ctx.Done():
		return Location{}, &LookupError{Class: FailureCanceled}
	}
	select {
	case result := <-call.result:
		return result.location, result.err
	case <-ctx.Done():
		return Location{}, &LookupError{Class: FailureCanceled}
	}
}

func (provider *controlledLookupProvider) nextCall(t *testing.T) *controlledLookupCall {
	t.Helper()
	select {
	case call := <-provider.calls:
		return call
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for provider call")
		return nil
	}
}

func (provider *controlledLookupProvider) assertNoCall(t *testing.T) {
	t.Helper()
	select {
	case call := <-provider.calls:
		t.Fatalf("unexpected provider call for %q", call.targetIP)
	case <-time.After(10 * time.Millisecond):
	}
}

func (call *controlledLookupCall) complete(location Location, err error) {
	call.result <- controlledLookupResult{location: location, err: err}
}

type manualCoordinatorRuntime struct {
	mu     sync.Mutex
	now    time.Time
	timers map[*manualCoordinatorTimer]struct{}
}

type manualCoordinatorTimer struct {
	runtime  *manualCoordinatorRuntime
	deadline time.Time
	channel  chan time.Time
	stopped  bool
	fired    bool
}

func newManualCoordinatorRuntime(now time.Time) *manualCoordinatorRuntime {
	return &manualCoordinatorRuntime{now: now.UTC(), timers: make(map[*manualCoordinatorTimer]struct{})}
}

func (runtime *manualCoordinatorRuntime) Now() time.Time {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return runtime.now
}

func (runtime *manualCoordinatorRuntime) NewTimer(delay time.Duration) coordinatorTimer {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	timer := &manualCoordinatorTimer{
		runtime:  runtime,
		deadline: runtime.now.Add(delay),
		channel:  make(chan time.Time, 1),
	}
	runtime.timers[timer] = struct{}{}
	return timer
}

func (runtime *manualCoordinatorRuntime) Advance(duration time.Duration) {
	runtime.mu.Lock()
	runtime.now = runtime.now.Add(duration)
	now := runtime.now
	due := make([]*manualCoordinatorTimer, 0)
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

func (runtime *manualCoordinatorRuntime) TimerCount() int {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return len(runtime.timers)
}

func (runtime *manualCoordinatorRuntime) waitForTimerCount(t *testing.T, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for runtime.TimerCount() != count {
		if time.Now().After(deadline) {
			t.Fatalf("timer count = %d, want %d", runtime.TimerCount(), count)
		}
		time.Sleep(time.Millisecond)
	}
}

func (timer *manualCoordinatorTimer) Channel() <-chan time.Time { return timer.channel }

func (timer *manualCoordinatorTimer) Stop() bool {
	timer.runtime.mu.Lock()
	defer timer.runtime.mu.Unlock()
	if timer.stopped || timer.fired {
		return false
	}
	timer.stopped = true
	delete(timer.runtime.timers, timer)
	return true
}
