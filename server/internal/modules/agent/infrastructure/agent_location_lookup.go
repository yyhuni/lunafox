package infrastructure

import (
	"context"
	"errors"

	"github.com/yyhuni/lunafox/server/internal/geolocation"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
)

type coordinatorAgentLocationLookup struct {
	coordinator *geolocation.Coordinator
}

// NewCoordinatorAgentLocationLookup adapts the shared provider coordinator to the Agent application port.
func NewCoordinatorAgentLocationLookup(coordinator *geolocation.Coordinator) (agentapp.AgentLocationLookup, error) {
	if coordinator == nil {
		return nil, errors.New("geolocation coordinator is required")
	}
	return &coordinatorAgentLocationLookup{coordinator: coordinator}, nil
}

func (lookup *coordinatorAgentLocationLookup) SubmitIP(
	ctx context.Context,
	sourceIP string,
	completion func(agentapp.AgentLocationLookupOutcome),
) error {
	if lookup == nil || lookup.coordinator == nil {
		return errors.New("geolocation coordinator is required")
	}
	if completion == nil {
		return errors.New("Agent location lookup completion is required")
	}
	return lookup.coordinator.SubmitIP(ctx, sourceIP, func(outcome geolocation.LookupOutcome) {
		mapped := agentapp.AgentLocationLookupOutcome{
			ProviderAttempted: outcome.ProviderAttempted,
			FromCache:         outcome.FromCache,
			Canceled:          outcome.Canceled,
			CompletedAt:       outcome.CompletedAt,
		}
		if outcome.Location != nil {
			mapped.Location = &agentapp.AgentLocationResolution{
				ObservedIP:       outcome.Location.ObservedIP,
				Latitude:         outcome.Location.Latitude,
				Longitude:        outcome.Location.Longitude,
				AccuracyRadiusKM: copyInfrastructureRadius(outcome.Location.AccuracyRadiusKM),
				ProviderKey:      outcome.Location.ProviderKey,
				ResolvedAt:       outcome.Location.ResolvedAt,
			}
		}
		if outcome.Failure != nil {
			mapped.FailureClass = string(outcome.Failure.Class)
		}
		completion(mapped)
	})
}

func copyInfrastructureRadius(radius *float64) *float64 {
	if radius == nil {
		return nil
	}
	copy := *radius
	return &copy
}
