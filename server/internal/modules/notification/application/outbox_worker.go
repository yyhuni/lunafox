package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	pkg "github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

const (
	defaultOutboxBatchSize = 50
	defaultOutboxLease     = time.Minute
	defaultOutboxPoll      = time.Second
)

// UnsupportedEventError separates deterministic envelope/template failures
// from transient database faults so only the former become terminal.
type UnsupportedEventError struct {
	Code string
	Err  error
}

func (err *UnsupportedEventError) Error() string {
	if err == nil || err.Err == nil {
		return "unsupported notification event"
	}
	return err.Err.Error()
}

func (err *UnsupportedEventError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Err
}

// OutboxMaterializer owns only durable fact/projection work. It must never
// make a provider request while its database transaction is open.
type OutboxMaterializer interface {
	Materialize(context.Context, domain.OutboxEvent) error
}

// OutboxWorkerOptions bounds polling and lease behavior for the supervised
// in-process worker.
type OutboxWorkerOptions struct {
	Owner         string
	PollInterval  time.Duration
	LeaseDuration time.Duration
	BatchSize     int
}

// OutboxWorker claims a bounded batch, materializes idempotently, and returns
// transient failures to the queue. It is intentionally independent of webhook
// delivery workers.
type OutboxWorker struct {
	store        OutboxStore
	materializer OutboxMaterializer
	options      OutboxWorkerOptions
	done         chan struct{}
	startOnce    sync.Once
}

func NewOutboxWorker(store OutboxStore, materializer OutboxMaterializer, options OutboxWorkerOptions) *OutboxWorker {
	if store == nil {
		panic("notification outbox store is required")
	}
	if materializer == nil {
		panic("notification outbox materializer is required")
	}
	if options.Owner == "" {
		panic("notification outbox worker owner is required")
	}
	if options.PollInterval <= 0 {
		options.PollInterval = defaultOutboxPoll
	}
	if options.LeaseDuration <= 0 {
		options.LeaseDuration = defaultOutboxLease
	}
	if options.BatchSize <= 0 {
		options.BatchSize = defaultOutboxBatchSize
	}
	return &OutboxWorker{store: store, materializer: materializer, options: options, done: make(chan struct{})}
}

func (worker *OutboxWorker) Start(ctx context.Context) {
	worker.startOnce.Do(func() {
		go worker.run(ctx)
	})
}

func (worker *OutboxWorker) Done() <-chan struct{} { return worker.done }

func (worker *OutboxWorker) run(ctx context.Context) {
	defer close(worker.done)
	ticker := time.NewTicker(worker.options.PollInterval)
	defer ticker.Stop()
	for {
		if err := worker.runOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			pkg.Warn("Notification outbox worker batch failed", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RunOnce is exposed for deterministic worker tests and performs no waiting.
func (worker *OutboxWorker) RunOnce(ctx context.Context) error { return worker.runOnce(ctx) }

func (worker *OutboxWorker) runOnce(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := time.Now().UTC()
	events, err := worker.store.Claim(ctx, worker.options.Owner, now, worker.options.LeaseDuration, worker.options.BatchSize)
	if err != nil {
		return fmt.Errorf("claim notification outbox: %w", err)
	}
	for _, event := range events {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := worker.materializer.Materialize(ctx, event); err != nil {
			var unsupported *UnsupportedEventError
			if errors.As(err, &unsupported) {
				code := unsupported.Code
				if code == "" {
					code = "unsupported_event"
				}
				if terminalErr := worker.store.MarkUnsupported(ctx, event.ID, worker.options.Owner, code, time.Now().UTC()); terminalErr != nil {
					return fmt.Errorf("mark unsupported event %d: %w", event.ID, terminalErr)
				}
				pkg.Error("Notification outbox event terminally unsupported",
					zap.Int64("notification.outbox_id", event.ID),
					zap.String("notification.event_id", event.Occurrence.EventID),
					zap.String("notification.kind", string(event.Occurrence.Kind)),
					zap.Int("notification.payload_version", event.Occurrence.PayloadVersion),
					zap.String("notification.failure_code", code),
				)
				continue
			}
			if releaseErr := worker.store.ReleaseLease(ctx, event.ID, worker.options.Owner, time.Now().UTC()); releaseErr != nil {
				return fmt.Errorf("release failed event %d after %w: %v", event.ID, err, releaseErr)
			}
			pkg.Warn("Notification outbox materialization deferred",
				zap.Int64("notification.outbox_id", event.ID),
				zap.String("notification.event_id", event.Occurrence.EventID),
				zap.Error(err),
			)
			continue
		}
		if err := worker.store.MarkPublished(ctx, event.ID, worker.options.Owner, time.Now().UTC()); err != nil {
			return fmt.Errorf("publish event %d: %w", event.ID, err)
		}
	}
	return nil
}
