package geolocation

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

const (
	coordinatorQueueCapacity = 128
	coordinatorMaxInFlight   = 4
	coordinatorMaxAttempts   = 60
	coordinatorAttemptWindow = time.Minute
	coordinatorCacheCapacity = 1024
	coordinatorSuccessTTL    = 7 * 24 * time.Hour
	coordinatorNegativeTTL   = 5 * time.Minute
)

var (
	ErrCoordinatorClosed    = errors.New("geolocation coordinator is closed")
	ErrCoordinatorQueueFull = errors.New("geolocation coordinator queue is full")
	ErrLookupCanceled       = errors.New("geolocation lookup submission is canceled")
	ErrCompletionRequired   = errors.New("geolocation lookup completion is required")
)

// LookupProvider resolves either a normalized public target IP or the provider-observed self IP.
type LookupProvider interface {
	Lookup(ctx context.Context, targetIP string) (Location, error)
}

// LookupOutcome is the normalized asynchronous completion delivered to a waiter.
type LookupOutcome struct {
	Location          *Location
	Failure           *LookupError
	ProviderAttempted bool
	FromCache         bool
	Canceled          bool
	CompletedAt       time.Time
}

// LookupCompletion receives one terminal outcome for an accepted live waiter.
type LookupCompletion func(LookupOutcome)

// Coordinator shares bounded provider resources, cache, and IP single-flight in one process.
type Coordinator struct {
	provider LookupProvider
	runtime  coordinatorRuntime
	options  coordinatorOptions

	mu            sync.Mutex
	queue         *lookupFlightDeque
	cache         *locationCache
	flightsByIP   map[string]*lookupFlight
	active        map[string]*lookupFlight
	attempts      []time.Time
	nextAllowedAt time.Time
	nextSelfID    uint64
	closing       bool

	rootContext context.Context
	cancelRoot  context.CancelFunc
	wake        chan struct{}
	results     chan providerCompletion
	done        chan struct{}
}

type coordinatorOptions struct {
	queueCapacity   int
	maxInFlight     int
	maxAttempts     int
	attemptWindow   time.Duration
	cacheCapacity   int
	successTTL      time.Duration
	negativeTTL     time.Duration
	attemptObserver func(providerAttemptEvent)
}

func defaultCoordinatorOptions() coordinatorOptions {
	return coordinatorOptions{
		queueCapacity:   coordinatorQueueCapacity,
		maxInFlight:     coordinatorMaxInFlight,
		maxAttempts:     coordinatorMaxAttempts,
		attemptWindow:   coordinatorAttemptWindow,
		cacheCapacity:   coordinatorCacheCapacity,
		successTTL:      coordinatorSuccessTTL,
		negativeTTL:     coordinatorNegativeTTL,
		attemptObserver: logProviderAttemptEvent,
	}
}

// NewCoordinator starts the fixed process-local FreeIPAPI resource coordinator.
func NewCoordinator(provider LookupProvider) (*Coordinator, error) {
	return newCoordinator(provider, realCoordinatorRuntime{}, defaultCoordinatorOptions())
}

func newCoordinator(provider LookupProvider, runtime coordinatorRuntime, options coordinatorOptions) (*Coordinator, error) {
	if provider == nil {
		return nil, fmt.Errorf("geolocation lookup provider is required")
	}
	if runtime == nil {
		return nil, fmt.Errorf("geolocation coordinator runtime is required")
	}
	if options.queueCapacity <= 0 || options.maxInFlight <= 0 || options.maxAttempts <= 0 || options.attemptWindow <= 0 || options.cacheCapacity <= 0 || options.successTTL <= 0 || options.negativeTTL <= 0 {
		return nil, fmt.Errorf("geolocation coordinator options must be positive")
	}
	rootContext, cancelRoot := context.WithCancel(context.Background())
	coordinator := &Coordinator{
		provider:    provider,
		runtime:     runtime,
		options:     options,
		queue:       newLookupFlightDeque(options.queueCapacity),
		cache:       newLocationCache(options.cacheCapacity, options.successTTL, options.negativeTTL),
		flightsByIP: make(map[string]*lookupFlight),
		active:      make(map[string]*lookupFlight),
		rootContext: rootContext,
		cancelRoot:  cancelRoot,
		wake:        make(chan struct{}, 1),
		results:     make(chan providerCompletion, options.maxInFlight),
		done:        make(chan struct{}),
	}
	go coordinator.run()
	return coordinator, nil
}

// SubmitIP admits a normalized public-IP lookup without waiting for provider capacity.
func (coordinator *Coordinator) SubmitIP(ctx context.Context, sourceIP string, completion LookupCompletion) error {
	if completion == nil {
		return ErrCompletionRequired
	}
	if ctx == nil || ctx.Err() != nil {
		return ErrLookupCanceled
	}
	normalizedIP, ok := NormalizePublicIP(sourceIP)
	if !ok {
		return &LookupError{Class: FailureInvalidTarget}
	}
	waiter := newLookupWaiter(ctx, completion)

	coordinator.mu.Lock()
	if coordinator.closing {
		coordinator.mu.Unlock()
		return ErrCoordinatorClosed
	}
	now := coordinator.runtime.Now().UTC()
	if cached, found := coordinator.cache.get(normalizedIP, now); found {
		coordinator.mu.Unlock()
		coordinator.deliverCached(waiter, cached, now)
		return nil
	}
	if flight, found := coordinator.flightsByIP[normalizedIP]; found {
		flight.waiters = append(flight.waiters, waiter)
		coordinator.mu.Unlock()
		coordinator.watchWaiter(waiter)
		return nil
	}
	if coordinator.queue.Full() {
		coordinator.mu.Unlock()
		return ErrCoordinatorQueueFull
	}
	flight := &lookupFlight{
		key:      "ip:" + normalizedIP,
		targetIP: normalizedIP,
		cacheIP:  normalizedIP,
		waiters:  []*lookupWaiter{waiter},
	}
	coordinator.flightsByIP[normalizedIP] = flight
	coordinator.queue.PushBack(flight)
	coordinator.mu.Unlock()
	coordinator.watchWaiter(waiter)
	coordinator.signal()
	return nil
}

// SubmitSelf admits one no-target provider lookup through the shared resource gates.
func (coordinator *Coordinator) SubmitSelf(ctx context.Context, completion LookupCompletion) error {
	if completion == nil {
		return ErrCompletionRequired
	}
	if ctx == nil || ctx.Err() != nil {
		return ErrLookupCanceled
	}
	waiter := newLookupWaiter(ctx, completion)
	coordinator.mu.Lock()
	if coordinator.closing {
		coordinator.mu.Unlock()
		return ErrCoordinatorClosed
	}
	if coordinator.queue.Full() {
		coordinator.mu.Unlock()
		return ErrCoordinatorQueueFull
	}
	coordinator.nextSelfID++
	flight := &lookupFlight{
		key:     fmt.Sprintf("self:%d", coordinator.nextSelfID),
		waiters: []*lookupWaiter{waiter},
	}
	coordinator.queue.PushBack(flight)
	coordinator.mu.Unlock()
	coordinator.watchWaiter(waiter)
	coordinator.signal()
	return nil
}

// NextAllowedAt returns the current monotonic provider-wide cooldown boundary.
func (coordinator *Coordinator) NextAllowedAt() time.Time {
	if coordinator == nil {
		return time.Time{}
	}
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	return coordinator.nextAllowedAt
}

// Shutdown closes admission, discards queued work, cancels active requests, and waits for exit.
func (coordinator *Coordinator) Shutdown(ctx context.Context) error {
	if coordinator == nil {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("shutdown context is required")
	}
	var discarded []*lookupFlight
	coordinator.mu.Lock()
	if !coordinator.closing {
		coordinator.closing = true
		discarded = coordinator.queue.Drain()
		for _, flight := range discarded {
			if flight.cacheIP != "" {
				delete(coordinator.flightsByIP, flight.cacheIP)
			}
		}
		coordinator.cancelRoot()
	}
	coordinator.mu.Unlock()
	for _, flight := range discarded {
		coordinator.deliverFlight(flight, LookupOutcome{Canceled: true, CompletedAt: coordinator.runtime.Now().UTC()})
	}
	coordinator.signal()

	select {
	case <-coordinator.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (coordinator *Coordinator) run() {
	defer close(coordinator.done)
	for {
		starts, delay, exit := coordinator.planStarts()
		for _, flight := range starts {
			go coordinator.execute(flight)
		}
		if exit {
			return
		}

		var timer coordinatorTimer
		var timerChannel <-chan time.Time
		if delay > 0 {
			timer = coordinator.runtime.NewTimer(delay)
			timerChannel = timer.Channel()
		}
		select {
		case completion := <-coordinator.results:
			if timer != nil {
				timer.Stop()
			}
			coordinator.handleProviderCompletion(completion)
		case <-coordinator.wake:
			if timer != nil {
				timer.Stop()
			}
		case <-timerChannel:
		}
	}
}

func (coordinator *Coordinator) planStarts() ([]*lookupFlight, time.Duration, bool) {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()

	now := coordinator.runtime.Now().UTC()
	coordinator.pruneCanceledQueue(now)
	for _, flight := range coordinator.active {
		if !flight.hasLiveWaiter() && flight.cancel != nil {
			flight.cancel()
		}
	}
	coordinator.pruneAttempts(now)
	starts := make([]*lookupFlight, 0, coordinator.options.maxInFlight-len(coordinator.active))
	for !coordinator.queue.Empty() && len(coordinator.active) < coordinator.options.maxInFlight {
		flight := coordinator.queue.Front()
		if !flight.hasLiveWaiter() {
			coordinator.queue.PopFront()
			if flight.cacheIP != "" {
				delete(coordinator.flightsByIP, flight.cacheIP)
			}
			flight.cancelWaiters(now)
			continue
		}
		if delay := coordinator.gateDelay(now); delay > 0 {
			return starts, delay, false
		}
		coordinator.queue.PopFront()
		flight.context, flight.cancel = context.WithCancel(coordinator.rootContext)
		flight.attemptedAt = now
		coordinator.active[flight.key] = flight
		coordinator.attempts = append(coordinator.attempts, now)
		starts = append(starts, flight)
	}
	return starts, 0, coordinator.closing && len(coordinator.active) == 0
}

func (coordinator *Coordinator) pruneCanceledQueue(now time.Time) {
	// Canceled waiters must release admission capacity even while every
	// execution permit is occupied; rotating the deque preserves FIFO order.
	queued := coordinator.queue.Len()
	for range queued {
		flight := coordinator.queue.PopFront()
		if flight.hasLiveWaiter() {
			coordinator.queue.PushBack(flight)
			continue
		}
		if flight.cacheIP != "" {
			delete(coordinator.flightsByIP, flight.cacheIP)
		}
		flight.cancelWaiters(now)
	}
}

func (coordinator *Coordinator) gateDelay(now time.Time) time.Duration {
	next := coordinator.nextAllowedAt
	if len(coordinator.attempts) >= coordinator.options.maxAttempts {
		rateNext := coordinator.attempts[0].Add(coordinator.options.attemptWindow)
		if rateNext.After(next) {
			next = rateNext
		}
	}
	if next.After(now) {
		return next.Sub(now)
	}
	return 0
}

func (coordinator *Coordinator) pruneAttempts(now time.Time) {
	cutoff := now.Add(-coordinator.options.attemptWindow)
	firstLive := 0
	for firstLive < len(coordinator.attempts) && !coordinator.attempts[firstLive].After(cutoff) {
		firstLive++
	}
	if firstLive > 0 {
		coordinator.attempts = append(coordinator.attempts[:0], coordinator.attempts[firstLive:]...)
	}
}

func (coordinator *Coordinator) execute(flight *lookupFlight) {
	location, err := coordinator.provider.Lookup(flight.context, flight.targetIP)
	coordinator.results <- providerCompletion{flight: flight, location: location, err: err}
}

func (coordinator *Coordinator) handleProviderCompletion(completion providerCompletion) {
	coordinator.mu.Lock()
	flight, active := coordinator.active[completion.flight.key]
	if !active || flight != completion.flight {
		coordinator.mu.Unlock()
		return
	}
	delete(coordinator.active, flight.key)
	if flight.cacheIP != "" {
		delete(coordinator.flightsByIP, flight.cacheIP)
	}
	if flight.cancel != nil {
		flight.cancel()
	}
	now := coordinator.runtime.Now().UTC()
	outcome := LookupOutcome{ProviderAttempted: true, CompletedAt: now}
	event := providerAttemptEvent{
		ProviderKey: FreeIPAPIProviderKey,
		ObservedIP:  flight.targetIP,
		AttemptedAt: flight.attemptedAt,
		Outcome:     "success",
	}
	if completion.err == nil {
		location := cloneLocation(completion.location)
		outcome.Location = &location
		if flight.cacheIP != "" {
			coordinator.cache.putSuccess(flight.cacheIP, location, now)
		}
	} else {
		failure := normalizedLookupError(completion.err)
		outcome.Failure = failure
		outcome.Canceled = failure.Class == FailureCanceled
		event.Outcome = "failed"
		event.FailureClass = failure.Class
		if failure.Class == FailureRateLimited && failure.RetryAfterAt.After(coordinator.nextAllowedAt) {
			coordinator.nextAllowedAt = failure.RetryAfterAt.UTC()
		}
		if flight.cacheIP != "" && failure.Class != FailureCanceled {
			coordinator.cache.putNegative(flight.cacheIP, failure.Class, now)
		}
	}
	if event.ObservedIP == "" && outcome.Location != nil {
		event.ObservedIP = outcome.Location.ObservedIP
	}
	attemptObserver := coordinator.options.attemptObserver
	coordinator.mu.Unlock()
	if attemptObserver != nil {
		attemptObserver(event)
	}
	coordinator.deliverFlight(flight, outcome)
	coordinator.signal()
}

func normalizedLookupError(err error) *LookupError {
	var failure *LookupError
	if errors.As(err, &failure) && failure != nil {
		copy := *failure
		return &copy
	}
	return &LookupError{Class: FailureNetwork}
}

func (coordinator *Coordinator) deliverCached(waiter *lookupWaiter, entry *locationCacheEntry, now time.Time) {
	outcome := LookupOutcome{FromCache: true, CompletedAt: now}
	if entry.kind == cacheEntrySuccess {
		location := cloneLocation(entry.location)
		outcome.Location = &location
	} else {
		outcome.Failure = &LookupError{Class: entry.failureClass}
		outcome.CompletedAt = entry.failedAt
	}
	coordinator.deliverWaiter(waiter, outcome)
}

func (coordinator *Coordinator) deliverFlight(flight *lookupFlight, outcome LookupOutcome) {
	for _, waiter := range flight.waiters {
		coordinator.deliverWaiter(waiter, outcome)
	}
}

func (coordinator *Coordinator) deliverWaiter(waiter *lookupWaiter, outcome LookupOutcome) {
	if waiter.context.Err() != nil {
		outcome = LookupOutcome{Canceled: true, CompletedAt: coordinator.runtime.Now().UTC()}
	}
	waiter.finish(func() {
		go waiter.completion(outcome)
	})
}

func (coordinator *Coordinator) watchWaiter(waiter *lookupWaiter) {
	if waiter.context.Done() == nil {
		return
	}
	go func() {
		select {
		case <-waiter.context.Done():
			coordinator.deliverWaiter(waiter, LookupOutcome{
				Canceled:    true,
				CompletedAt: coordinator.runtime.Now().UTC(),
			})
			coordinator.signal()
		case <-waiter.done:
		}
	}()
}

func (coordinator *Coordinator) signal() {
	select {
	case coordinator.wake <- struct{}{}:
	default:
	}
}

type lookupWaiter struct {
	context    context.Context
	completion LookupCompletion
	done       chan struct{}
	finishOnce sync.Once
}

func newLookupWaiter(ctx context.Context, completion LookupCompletion) *lookupWaiter {
	return &lookupWaiter{context: ctx, completion: completion, done: make(chan struct{})}
}

func (waiter *lookupWaiter) finish(beforeClose func()) {
	waiter.finishOnce.Do(func() {
		if beforeClose != nil {
			beforeClose()
		}
		close(waiter.done)
	})
}

type lookupFlight struct {
	key         string
	targetIP    string
	cacheIP     string
	waiters     []*lookupWaiter
	context     context.Context
	cancel      context.CancelFunc
	attemptedAt time.Time
}

func (flight *lookupFlight) hasLiveWaiter() bool {
	for _, waiter := range flight.waiters {
		if waiter.context.Err() == nil {
			return true
		}
	}
	return false
}

func (flight *lookupFlight) cancelWaiters(completedAt time.Time) {
	for _, waiter := range flight.waiters {
		if waiter.context.Err() != nil {
			waiter.finish(func() {
				go waiter.completion(LookupOutcome{Canceled: true, CompletedAt: completedAt})
			})
		}
	}
}

type providerCompletion struct {
	flight   *lookupFlight
	location Location
	err      error
}

type lookupFlightDeque struct {
	entries []*lookupFlight
	head    int
	size    int
}

func newLookupFlightDeque(capacity int) *lookupFlightDeque {
	return &lookupFlightDeque{entries: make([]*lookupFlight, capacity)}
}

func (deque *lookupFlightDeque) Full() bool  { return deque.size == len(deque.entries) }
func (deque *lookupFlightDeque) Empty() bool { return deque.size == 0 }
func (deque *lookupFlightDeque) Len() int    { return deque.size }

func (deque *lookupFlightDeque) PushBack(flight *lookupFlight) {
	index := (deque.head + deque.size) % len(deque.entries)
	deque.entries[index] = flight
	deque.size++
}

func (deque *lookupFlightDeque) Front() *lookupFlight {
	if deque.size == 0 {
		return nil
	}
	return deque.entries[deque.head]
}

func (deque *lookupFlightDeque) PopFront() *lookupFlight {
	if deque.size == 0 {
		return nil
	}
	flight := deque.entries[deque.head]
	deque.entries[deque.head] = nil
	deque.head = (deque.head + 1) % len(deque.entries)
	deque.size--
	return flight
}

func (deque *lookupFlightDeque) Drain() []*lookupFlight {
	flights := make([]*lookupFlight, 0, deque.size)
	for !deque.Empty() {
		flights = append(flights, deque.PopFront())
	}
	return flights
}

type coordinatorRuntime interface {
	Now() time.Time
	NewTimer(delay time.Duration) coordinatorTimer
}

type coordinatorTimer interface {
	Channel() <-chan time.Time
	Stop() bool
}

type realCoordinatorRuntime struct{}

func (realCoordinatorRuntime) Now() time.Time { return time.Now().UTC() }
func (realCoordinatorRuntime) NewTimer(delay time.Duration) coordinatorTimer {
	return realCoordinatorTimer{timer: time.NewTimer(delay)}
}

type realCoordinatorTimer struct {
	timer *time.Timer
}

func (timer realCoordinatorTimer) Channel() <-chan time.Time { return timer.timer.C }
func (timer realCoordinatorTimer) Stop() bool                { return timer.timer.Stop() }

type coordinatorSnapshot struct {
	Queued        int
	Active        int
	Flights       int
	CacheEntries  int
	Attempts      int
	NextAllowedAt time.Time
	Closing       bool
}

type providerAttemptEvent struct {
	ProviderKey  string
	ObservedIP   string
	AttemptedAt  time.Time
	Outcome      string
	FailureClass FailureClass
}

func logProviderAttemptEvent(event providerAttemptEvent) {
	fields := []zap.Field{
		zap.String("providerKey", event.ProviderKey),
		zap.Time("attemptedAt", event.AttemptedAt.UTC()),
		zap.String("outcome", event.Outcome),
	}
	if event.FailureClass == "" {
		pkg.Info("GeoIP provider attempt completed", fields...)
		return
	}
	fields = append(fields, zap.String("failureClass", string(event.FailureClass)))
	pkg.Warn("GeoIP provider attempt failed", fields...)
}

func (coordinator *Coordinator) snapshot() coordinatorSnapshot {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	return coordinatorSnapshot{
		Queued:        coordinator.queue.Len(),
		Active:        len(coordinator.active),
		Flights:       len(coordinator.flightsByIP),
		CacheEntries:  len(coordinator.cache.entries),
		Attempts:      len(coordinator.attempts),
		NextAllowedAt: coordinator.nextAllowedAt,
		Closing:       coordinator.closing,
	}
}
