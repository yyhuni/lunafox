package application

import (
	"context"
	"errors"
	"sync"

	"github.com/yyhuni/lunafox/server/internal/geolocation"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

// AgentLocationObserver evaluates and persists best-effort GeoIP enhancement after SessionReady.
type AgentLocationObserver struct {
	repository agentdomain.AgentLocationRepository
	lookup     AgentLocationLookup
	clock      Clock

	context context.Context
	cancel  context.CancelFunc

	mu           sync.Mutex
	closed       bool
	pending      sync.WaitGroup
	shutdownOnce sync.Once
	done         chan struct{}
}

// NewAgentLocationObserver constructs the non-blocking post-SessionReady observer.
func NewAgentLocationObserver(
	repository agentdomain.AgentLocationRepository,
	lookup AgentLocationLookup,
	clock Clock,
) (*AgentLocationObserver, error) {
	if repository == nil {
		return nil, errors.New("Agent location repository is required")
	}
	if lookup == nil {
		return nil, errors.New("Agent location lookup is required")
	}
	if clock == nil {
		return nil, errors.New("Agent location clock is required")
	}
	observerContext, cancel := context.WithCancel(context.Background())
	return &AgentLocationObserver{
		repository: repository,
		lookup:     lookup,
		clock:      clock,
		context:    observerContext,
		cancel:     cancel,
		done:       make(chan struct{}),
	}, nil
}

// ObserveReadyConnection submits eligible location work without returning enhancement failures to gRPC.
func (observer *AgentLocationObserver) ObserveReadyConnection(agent *agentdomain.Agent) {
	if observer == nil || agent == nil || agent.ID <= 0 || agent.ObservedIPGeneration <= 0 {
		return
	}
	normalizedSource, ok := geolocation.NormalizePublicIP(agent.ObservedSourceIP)
	if !ok {
		return
	}
	now := observer.clock.NowUTC()
	if snapshot := agent.Location; snapshot != nil && snapshot.SourceObservedIP == normalizedSource && !snapshot.IsExpiredAt(now) {
		return
	}

	waiter := agentLocationWaiter{
		agentID:    agent.ID,
		sourceIP:   normalizedSource,
		generation: agent.ObservedIPGeneration,
	}
	observer.mu.Lock()
	if observer.closed {
		observer.mu.Unlock()
		return
	}
	observer.pending.Add(1)
	lookupContext := observer.context
	observer.mu.Unlock()

	err := observer.lookup.SubmitIP(lookupContext, normalizedSource, func(outcome AgentLocationLookupOutcome) {
		defer observer.pending.Done()
		observer.persistOutcome(waiter, outcome)
	})
	if err != nil {
		observer.pending.Done()
		pkg.Warn("Agent location lookup submission rejected",
			zap.Int("agent.id", agent.ID),
			zap.String("failureClass", "submission_rejected"),
		)
	}
}

// Shutdown closes observation admission, cancels accepted waiters, and waits for persistence callbacks.
func (observer *AgentLocationObserver) Shutdown(ctx context.Context) error {
	if observer == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("Agent location observer shutdown context is required")
	}
	observer.shutdownOnce.Do(func() {
		observer.mu.Lock()
		observer.closed = true
		observer.cancel()
		observer.mu.Unlock()
		go func() {
			observer.pending.Wait()
			close(observer.done)
		}()
	})
	select {
	case <-observer.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type agentLocationWaiter struct {
	agentID    int
	sourceIP   string
	generation int64
}

func (observer *AgentLocationObserver) persistOutcome(waiter agentLocationWaiter, outcome AgentLocationLookupOutcome) {
	if outcome.Canceled || observer.context.Err() != nil {
		return
	}
	if outcome.Location != nil {
		location := outcome.Location
		if location.ObservedIP != waiter.sourceIP {
			pkg.Warn("Agent location result source mismatch",
				zap.Int("agent.id", waiter.agentID),
				zap.String("failureClass", "source_mismatch"),
			)
			return
		}
		snapshot := agentdomain.AgentLocationSnapshot{
			AgentID:          waiter.agentID,
			Latitude:         location.Latitude,
			Longitude:        location.Longitude,
			AccuracyRadiusKM: copyAgentLocationRadius(location.AccuracyRadiusKM),
			SourceObservedIP: waiter.sourceIP,
			ProviderKey:      location.ProviderKey,
			ResolvedAt:       location.ResolvedAt.UTC(),
			UpdatedAt:        observer.clock.NowUTC(),
		}
		if _, err := observer.repository.ReplaceLocationIfObservationMatches(observer.context, snapshot, waiter.generation); err != nil && observer.context.Err() == nil {
			pkg.Warn("Agent location success could not be persisted",
				zap.Int("agent.id", waiter.agentID),
				zap.Error(err),
			)
		}
		return
	}
	if outcome.FailureClass == "" {
		pkg.Warn("Agent location lookup returned an incomplete terminal outcome",
			zap.Int("agent.id", waiter.agentID),
			zap.String("failureClass", "invalid_outcome"),
		)
		return
	}
	if _, err := observer.repository.MarkLocationExpiredIfObservationMatches(observer.context, waiter.agentID, waiter.sourceIP, waiter.generation); err != nil && observer.context.Err() == nil {
		pkg.Warn("Agent location failure state could not be persisted",
			zap.Int("agent.id", waiter.agentID),
			zap.String("failureClass", outcome.FailureClass),
			zap.Error(err),
		)
	}
}

func copyAgentLocationRadius(radius *float64) *float64 {
	if radius == nil {
		return nil
	}
	copy := *radius
	return &copy
}
