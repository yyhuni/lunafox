package application

import (
	"context"
	"time"
)

// AgentLocationResolution is the provider-neutral successful lookup payload used by Agent application code.
type AgentLocationResolution struct {
	ObservedIP       string
	Latitude         float64
	Longitude        float64
	AccuracyRadiusKM *float64
	ProviderKey      string
	ResolvedAt       time.Time
}

// AgentLocationLookupOutcome is one terminal result for an accepted lookup waiter.
type AgentLocationLookupOutcome struct {
	Location          *AgentLocationResolution
	FailureClass      string
	ProviderAttempted bool
	FromCache         bool
	Canceled          bool
	CompletedAt       time.Time
}

// AgentLocationLookup submits bounded asynchronous public-IP resolution work.
type AgentLocationLookup interface {
	SubmitIP(ctx context.Context, sourceIP string, completion func(AgentLocationLookupOutcome)) error
}
