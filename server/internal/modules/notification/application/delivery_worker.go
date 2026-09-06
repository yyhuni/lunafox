package application

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	pkg "github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

const (
	defaultDeliveryBatchSize = 25
	defaultDeliveryLease     = 30 * time.Second
	defaultDeliveryPoll      = time.Second
	maxDeliveryAttempts      = 6
	deliveryRetryWindow      = time.Hour
	initialRetryBackoff      = 30 * time.Second
	maximumRetryBackoff      = 15 * time.Minute
)

// DeliveryWorkerOptions bounds durable external delivery processing.
type DeliveryWorkerOptions struct {
	Owner         string
	PollInterval  time.Duration
	LeaseDuration time.Duration
	BatchSize     int
}

// DeliveryWorker claims durable records, executes one HTTP attempt outside any
// database transaction, and persists only its redacted outcome.
type DeliveryWorker struct {
	store        DeliveryStore
	destinations DestinationStore
	adapters     map[domain.Provider]ProviderAdapter
	options      DeliveryWorkerOptions
	done         chan struct{}
	startOnce    sync.Once
	now          func() time.Time
	jitter       func(time.Duration) time.Duration
}

// NewDeliveryWorker creates a supervised delivery processor.
func NewDeliveryWorker(store DeliveryStore, destinations DestinationStore, adapters []ProviderAdapter, options DeliveryWorkerOptions) *DeliveryWorker {
	if store == nil || destinations == nil {
		panic("notification delivery worker dependencies are required")
	}
	if strings.TrimSpace(options.Owner) == "" {
		panic("notification delivery worker owner is required")
	}
	if options.PollInterval <= 0 {
		options.PollInterval = defaultDeliveryPoll
	}
	if options.LeaseDuration <= 0 {
		options.LeaseDuration = defaultDeliveryLease
	}
	if options.BatchSize <= 0 {
		options.BatchSize = defaultDeliveryBatchSize
	}
	adapterMap, err := newProviderAdapterRegistry(adapters)
	if err != nil {
		panic(err)
	}
	return &DeliveryWorker{
		store:        store,
		destinations: destinations,
		adapters:     adapterMap,
		options:      options,
		done:         make(chan struct{}),
		now:          func() time.Time { return time.Now().UTC() },
		jitter:       fullJitter,
	}
}

// Start begins bounded polling. Cancellation stops new claims; an attempt
// already begun gets its own timeout-bounded context in processDelivery.
func (worker *DeliveryWorker) Start(ctx context.Context) {
	worker.startOnce.Do(func() {
		go worker.run(ctx)
	})
}

// Done closes after the worker has stopped claiming new delivery work.
func (worker *DeliveryWorker) Done() <-chan struct{} { return worker.done }

func (worker *DeliveryWorker) run(ctx context.Context) {
	defer close(worker.done)
	ticker := time.NewTicker(worker.options.PollInterval)
	defer ticker.Stop()
	for {
		if err := worker.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			pkg.Warn("Notification delivery worker batch failed", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RunOnce performs a deterministic bounded claim cycle without sleeping.
func (worker *DeliveryWorker) RunOnce(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := worker.now().UTC()
	deliveries, err := worker.store.Claim(ctx, worker.options.Owner, now, worker.options.LeaseDuration, worker.options.BatchSize)
	if err != nil {
		return fmt.Errorf("claim notification deliveries: %w", err)
	}
	for _, delivery := range deliveries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := worker.processDelivery(ctx, delivery); err != nil {
			return err
		}
	}
	return nil
}

func (worker *DeliveryWorker) processDelivery(ctx context.Context, delivery domain.Delivery) error {
	destination, err := worker.destinations.Get(ctx, delivery.Provider)
	if err != nil {
		return worker.releaseTransientDelivery(ctx, delivery, fmt.Errorf("load destination: %w", err))
	}
	if err := domain.ValidateDestination(destination); err != nil || !destination.Enabled {
		return worker.failWithoutAttempt(ctx, delivery, "destination_configuration")
	}
	adapter, found := worker.adapters[delivery.Provider]
	if !found {
		return worker.failWithoutAttempt(ctx, delivery, "unsupported_provider")
	}

	startedAt := worker.now().UTC()
	// A graceful job-context cancellation stops the next claim, but this active
	// request retains its ten-second adapter deadline before lease recovery.
	attemptContext := context.WithoutCancel(ctx)
	result := adapter.Deliver(attemptContext, destination, delivery)
	completedAt := worker.now().UTC()
	attempt := domain.DeliveryAttempt{
		DeliveryID:        delivery.ID,
		AttemptNumber:     delivery.AttemptCount + 1,
		AttemptedAt:       startedAt,
		CompletedAt:       completedAt,
		HTTPStatus:        result.HTTPStatus,
		ErrorClass:        safeErrorClass(result.ErrorClass),
		ProviderRequestID: safeProviderRequestID(result.ProviderRequestID),
	}
	if err := worker.store.RecordAttempt(attemptContext, attempt, worker.options.Owner); err != nil {
		return fmt.Errorf("record notification delivery attempt %d: %w", delivery.ID, err)
	}
	worker.logAttempt(delivery, attempt, result)

	if result.Accepted {
		if err := worker.store.MarkDelivered(attemptContext, delivery.ID, worker.options.Owner, completedAt); err != nil {
			return fmt.Errorf("mark notification delivery %d delivered: %w", delivery.ID, err)
		}
		pkg.Info("Notification delivery completed",
			zap.Int64("notification.delivery_id", delivery.ID),
			zap.String("notification.event_id", delivery.EventID),
			zap.String("notification.provider", string(delivery.Provider)),
			zap.Int("notification.attempt", attempt.AttemptNumber),
			zap.Time("notification.completed_at", completedAt),
		)
		return nil
	}

	if nextAttemptAt, retry := worker.nextRetryAt(delivery, attempt, result, completedAt); retry {
		if err := worker.store.MarkRetry(attemptContext, delivery.ID, worker.options.Owner, nextAttemptAt, completedAt); err != nil {
			return fmt.Errorf("schedule notification delivery %d retry: %w", delivery.ID, err)
		}
		return nil
	}
	if err := worker.store.MarkFailedTerminal(attemptContext, delivery.ID, worker.options.Owner, completedAt); err != nil {
		return fmt.Errorf("mark notification delivery %d terminal: %w", delivery.ID, err)
	}
	worker.logTerminalFailure(delivery, attempt, completedAt)
	return nil
}

func (worker *DeliveryWorker) releaseTransientDelivery(ctx context.Context, delivery domain.Delivery, cause error) error {
	// Destination reads are database work; a transient repository failure is not
	// evidence that the destination is invalid, so leave the lease for recovery.
	pkg.Warn("Notification delivery destination lookup deferred",
		zap.Int64("notification.delivery_id", delivery.ID),
		zap.String("notification.event_id", delivery.EventID),
		zap.Error(cause),
	)
	return worker.store.MarkRetry(ctx, delivery.ID, worker.options.Owner, worker.now().UTC(), worker.now().UTC())
}

func (worker *DeliveryWorker) failWithoutAttempt(ctx context.Context, delivery domain.Delivery, errorClass string) error {
	now := worker.now().UTC()
	if err := worker.store.MarkFailedTerminal(ctx, delivery.ID, worker.options.Owner, now); err != nil {
		return fmt.Errorf("mark notification delivery %d configuration terminal: %w", delivery.ID, err)
	}
	pkg.Error("Notification delivery terminally rejected before attempt",
		zap.Int64("notification.delivery_id", delivery.ID),
		zap.String("notification.event_id", delivery.EventID),
		zap.String("notification.provider", string(delivery.Provider)),
		zap.Int("notification.attempt", delivery.AttemptCount),
		zap.Time("notification.completed_at", now),
		zap.String("notification.error_class", errorClass),
	)
	return nil
}

func (worker *DeliveryWorker) nextRetryAt(delivery domain.Delivery, attempt domain.DeliveryAttempt, result DeliveryResult, now time.Time) (time.Time, bool) {
	if !result.Retryable || attempt.AttemptNumber >= maxDeliveryAttempts {
		return time.Time{}, false
	}
	firstAttemptAt := delivery.FirstAttemptAt
	if firstAttemptAt == nil {
		firstAttemptAt = &attempt.AttemptedAt
	}
	deadline := firstAttemptAt.UTC().Add(deliveryRetryWindow)
	if !now.Before(deadline) {
		return time.Time{}, false
	}
	if result.RetryAfter != nil && result.RetryAfter.After(now) {
		if !result.RetryAfter.Before(deadline) {
			return time.Time{}, false
		}
		return result.RetryAfter.UTC(), true
	}
	nextAttemptAt := now.Add(worker.jitter(backoffCap(attempt.AttemptNumber)))
	if !nextAttemptAt.Before(deadline) {
		return time.Time{}, false
	}
	return nextAttemptAt, true
}

func backoffCap(attemptNumber int) time.Duration {
	if attemptNumber <= 1 {
		return initialRetryBackoff
	}
	backoff := initialRetryBackoff
	for step := 1; step < attemptNumber && backoff < maximumRetryBackoff; step++ {
		backoff *= 2
		if backoff > maximumRetryBackoff {
			return maximumRetryBackoff
		}
	}
	return backoff
}

func fullJitter(cap time.Duration) time.Duration {
	if cap <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(cap) + 1))
}

func (worker *DeliveryWorker) logAttempt(delivery domain.Delivery, attempt domain.DeliveryAttempt, result DeliveryResult) {
	fields := []zap.Field{
		zap.Int64("notification.delivery_id", delivery.ID),
		zap.String("notification.event_id", delivery.EventID),
		zap.String("notification.provider", string(delivery.Provider)),
		zap.Int("notification.attempt", attempt.AttemptNumber),
		zap.Time("notification.attempted_at", attempt.AttemptedAt),
		zap.Time("notification.completed_at", attempt.CompletedAt),
		zap.String("notification.error_class", attempt.ErrorClass),
		zap.String("notification.provider_request_id", attempt.ProviderRequestID),
	}
	if attempt.HTTPStatus != nil {
		fields = append(fields, zap.Int("notification.http_status", *attempt.HTTPStatus))
	}
	if result.Accepted {
		pkg.Info("Notification delivery attempt accepted", fields...)
		return
	}
	pkg.Warn("Notification delivery attempt failed", fields...)
}

func (worker *DeliveryWorker) logTerminalFailure(delivery domain.Delivery, attempt domain.DeliveryAttempt, terminalAt time.Time) {
	pkg.Error("Notification delivery terminally failed",
		zap.Int64("notification.delivery_id", delivery.ID),
		zap.String("notification.event_id", delivery.EventID),
		zap.String("notification.provider", string(delivery.Provider)),
		zap.Int("notification.attempt", attempt.AttemptNumber),
		zap.Time("notification.terminal_at", terminalAt),
		zap.String("notification.error_class", attempt.ErrorClass),
		zap.String("notification.provider_request_id", attempt.ProviderRequestID),
	)
}

func safeErrorClass(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "provider_outcome_unknown"
	}
	if len(raw) > 128 {
		return "provider_error_class_truncated"
	}
	return raw
}

func safeProviderRequestID(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) > 500 || strings.Contains(raw, "://") || strings.Contains(raw, "@") {
		return ""
	}
	return raw
}
