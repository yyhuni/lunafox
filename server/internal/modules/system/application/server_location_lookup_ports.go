package application

import (
	"context"
	"time"
)

type ServerLocationResolution struct {
	ObservedEgressIP string
	Latitude         float64
	Longitude        float64
	AccuracyRadiusKM *float64
	ProviderKey      string
	ResolvedAt       time.Time
}

type ServerLocationLookupOutcome struct {
	Location          *ServerLocationResolution
	FailureClass      string
	ProviderAttempted bool
	Canceled          bool
	CompletedAt       time.Time
}

// ServerLocationLookup submits no-target provider work through shared process gates.
type ServerLocationLookup interface {
	SubmitSelf(ctx context.Context, completion func(ServerLocationLookupOutcome)) error
	NextAllowedAt() time.Time
}

// ServerLocationSchedulerRuntime provides the one-shot timer and clock used by the lifecycle.
type ServerLocationSchedulerRuntime interface {
	ServerLocationClock
	NewTimer(delay time.Duration) ServerLocationTimer
}

type ServerLocationTimer interface {
	Channel() <-chan time.Time
	Stop() bool
}
