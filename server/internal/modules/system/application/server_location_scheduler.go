package application

import (
	"context"
	"errors"
	"sync"
	"time"

	systemdomain "github.com/yyhuni/lunafox/server/internal/modules/system/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

const ServerLocationFailureRetryDelay = 5 * time.Minute

// ServerLocationScheduler owns the single Server self-location refresh lifecycle.
type ServerLocationScheduler struct {
	repository systemdomain.ServerLocationRepository
	lookup     ServerLocationLookup
	runtime    ServerLocationSchedulerRuntime

	mu      sync.Mutex
	started bool
	done    chan struct{}
}

func NewServerLocationScheduler(
	repository systemdomain.ServerLocationRepository,
	lookup ServerLocationLookup,
	runtime ServerLocationSchedulerRuntime,
) (*ServerLocationScheduler, error) {
	if repository == nil {
		return nil, errors.New("Server location repository is required")
	}
	if lookup == nil {
		return nil, errors.New("Server location lookup is required")
	}
	if runtime == nil {
		return nil, errors.New("Server location scheduler runtime is required")
	}
	return &ServerLocationScheduler{
		repository: repository,
		lookup:     lookup,
		runtime:    runtime,
		done:       make(chan struct{}),
	}, nil
}

func (scheduler *ServerLocationScheduler) Start(ctx context.Context) {
	if scheduler == nil {
		return
	}
	if ctx == nil {
		panic("Server location scheduler context is required")
	}
	scheduler.mu.Lock()
	if scheduler.started {
		scheduler.mu.Unlock()
		return
	}
	scheduler.started = true
	scheduler.mu.Unlock()
	go scheduler.run(ctx)
}

func (scheduler *ServerLocationScheduler) Done() <-chan struct{} {
	if scheduler == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	return scheduler.done
}

func (scheduler *ServerLocationScheduler) run(ctx context.Context) {
	defer close(scheduler.done)
	snapshot, ok := scheduler.restoreSnapshot(ctx)
	if !ok {
		return
	}

	nextAttemptAt := scheduler.runtime.NowUTC()
	if snapshot != nil && !snapshot.IsExpiredAt(nextAttemptAt) {
		nextAttemptAt = snapshot.ResolvedAt.UTC().Add(systemdomain.ServerLocationFreshness)
	}
	for {
		if !scheduler.waitUntil(ctx, nextAttemptAt) {
			return
		}
		outcome, submitted := scheduler.submitAndWait(ctx)
		if !submitted {
			if ctx.Err() != nil {
				return
			}
			nextAttemptAt = scheduler.retryAt(scheduler.runtime.NowUTC())
			continue
		}
		if outcome.Canceled {
			if ctx.Err() != nil {
				return
			}
			nextAttemptAt = scheduler.retryAt(scheduler.runtime.NowUTC())
			continue
		}
		if outcome.Location == nil {
			failedAt := outcome.CompletedAt
			if failedAt.IsZero() {
				failedAt = scheduler.runtime.NowUTC()
			}
			nextAttemptAt = scheduler.retryAt(failedAt)
			continue
		}

		location := outcome.Location
		replacement := systemdomain.ServerLocationSnapshot{
			ObservedEgressIP: location.ObservedEgressIP,
			Latitude:         location.Latitude,
			Longitude:        location.Longitude,
			AccuracyRadiusKM: copyScheduledServerLocationRadius(location.AccuracyRadiusKM),
			ProviderKey:      location.ProviderKey,
			ResolvedAt:       location.ResolvedAt.UTC(),
			UpdatedAt:        scheduler.runtime.NowUTC(),
		}
		if err := scheduler.repository.Replace(ctx, replacement); err != nil {
			if ctx.Err() != nil {
				return
			}
			pkg.Warn("Server location success could not be persisted", zap.Error(err))
			nextAttemptAt = scheduler.retryAt(scheduler.runtime.NowUTC())
			continue
		}
		snapshot = &replacement
		nextAttemptAt = snapshot.ResolvedAt.Add(systemdomain.ServerLocationFreshness)
	}
}

func (scheduler *ServerLocationScheduler) restoreSnapshot(ctx context.Context) (*systemdomain.ServerLocationSnapshot, bool) {
	for {
		snapshot, err := scheduler.repository.Get(ctx)
		if err == nil {
			return snapshot, true
		}
		if ctx.Err() != nil {
			return nil, false
		}
		pkg.Warn("Server location snapshot could not be restored", zap.Error(err))
		if !scheduler.waitUntil(ctx, scheduler.runtime.NowUTC().Add(ServerLocationFailureRetryDelay)) {
			return nil, false
		}
	}
}

func (scheduler *ServerLocationScheduler) submitAndWait(ctx context.Context) (ServerLocationLookupOutcome, bool) {
	results := make(chan ServerLocationLookupOutcome, 1)
	err := scheduler.lookup.SubmitSelf(ctx, func(outcome ServerLocationLookupOutcome) {
		select {
		case results <- outcome:
		case <-ctx.Done():
		}
	})
	if err != nil {
		pkg.Warn("Server location lookup submission rejected",
			zap.String("failureClass", "submission_rejected"),
		)
		return ServerLocationLookupOutcome{}, false
	}
	select {
	case outcome := <-results:
		return outcome, true
	case <-ctx.Done():
		return ServerLocationLookupOutcome{Canceled: true, CompletedAt: scheduler.runtime.NowUTC()}, true
	}
}

func (scheduler *ServerLocationScheduler) retryAt(failedAt time.Time) time.Time {
	next := failedAt.UTC().Add(ServerLocationFailureRetryDelay)
	if cooldown := scheduler.lookup.NextAllowedAt().UTC(); cooldown.After(next) {
		next = cooldown
	}
	if now := scheduler.runtime.NowUTC(); now.After(next) {
		return now
	}
	return next
}

func (scheduler *ServerLocationScheduler) waitUntil(ctx context.Context, instant time.Time) bool {
	delay := instant.Sub(scheduler.runtime.NowUTC())
	if delay <= 0 {
		return ctx.Err() == nil
	}
	timer := scheduler.runtime.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.Channel():
		return ctx.Err() == nil
	case <-ctx.Done():
		return false
	}
}

func copyScheduledServerLocationRadius(radius *float64) *float64 {
	if radius == nil {
		return nil
	}
	copy := *radius
	return &copy
}
