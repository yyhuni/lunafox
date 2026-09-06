package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/yyhuni/lunafox/server/internal/geolocation"
	systemapp "github.com/yyhuni/lunafox/server/internal/modules/system/application"
)

type coordinatorServerLocationLookup struct {
	coordinator *geolocation.Coordinator
}

func NewCoordinatorServerLocationLookup(coordinator *geolocation.Coordinator) (systemapp.ServerLocationLookup, error) {
	if coordinator == nil {
		return nil, errors.New("geolocation coordinator is required")
	}
	return &coordinatorServerLocationLookup{coordinator: coordinator}, nil
}

func (lookup *coordinatorServerLocationLookup) SubmitSelf(
	ctx context.Context,
	completion func(systemapp.ServerLocationLookupOutcome),
) error {
	if lookup == nil || lookup.coordinator == nil {
		return errors.New("geolocation coordinator is required")
	}
	if completion == nil {
		return errors.New("Server location lookup completion is required")
	}
	return lookup.coordinator.SubmitSelf(ctx, func(outcome geolocation.LookupOutcome) {
		mapped := systemapp.ServerLocationLookupOutcome{
			ProviderAttempted: outcome.ProviderAttempted,
			Canceled:          outcome.Canceled,
			CompletedAt:       outcome.CompletedAt,
		}
		if outcome.Location != nil {
			mapped.Location = &systemapp.ServerLocationResolution{
				ObservedEgressIP: outcome.Location.ObservedIP,
				Latitude:         outcome.Location.Latitude,
				Longitude:        outcome.Location.Longitude,
				AccuracyRadiusKM: copySystemLocationRadius(outcome.Location.AccuracyRadiusKM),
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

func (lookup *coordinatorServerLocationLookup) NextAllowedAt() time.Time {
	if lookup == nil || lookup.coordinator == nil {
		return time.Time{}
	}
	return lookup.coordinator.NextAllowedAt()
}

func copySystemLocationRadius(radius *float64) *float64 {
	if radius == nil {
		return nil
	}
	copy := *radius
	return &copy
}
