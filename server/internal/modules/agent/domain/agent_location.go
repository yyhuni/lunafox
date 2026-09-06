package domain

import (
	"context"
	"time"
)

const AgentLocationFreshness = 7 * 24 * time.Hour

type AgentLocationState string

const (
	AgentLocationStateUnknown AgentLocationState = "unknown"
	AgentLocationStateCurrent AgentLocationState = "current"
	AgentLocationStateExpired AgentLocationState = "expired"
)

// AgentLocationSnapshot binds coordinates to the observed source that produced them.
type AgentLocationSnapshot struct {
	AgentID          int
	Latitude         float64
	Longitude        float64
	AccuracyRadiusKM *float64
	SourceObservedIP string
	ProviderKey      string
	ResolvedAt       time.Time
	ForcedExpired    bool
	UpdatedAt        time.Time
}

func (snapshot AgentLocationSnapshot) IsExpiredAt(now time.Time) bool {
	return snapshot.ForcedExpired || !now.Before(snapshot.ResolvedAt.Add(AgentLocationFreshness))
}

func (snapshot *AgentLocationSnapshot) StateAt(now time.Time) AgentLocationState {
	if snapshot == nil {
		return AgentLocationStateUnknown
	}
	if snapshot.IsExpiredAt(now.UTC()) {
		return AgentLocationStateExpired
	}
	return AgentLocationStateCurrent
}

// AgentConnectionObservation is the persisted source fence for one ready control session.
type AgentConnectionObservation struct {
	AgentID           int
	ConnectionIP      string
	SourceIP          string
	Generation        int64
	SourceChanged     bool
	ConnectionChanged bool
}

type AgentLocationRepository interface {
	GetLocation(ctx context.Context, agentID int) (*AgentLocationSnapshot, error)
	ReplaceLocationIfObservationMatches(ctx context.Context, snapshot AgentLocationSnapshot, generation int64) (bool, error)
	MarkLocationExpiredIfObservationMatches(ctx context.Context, agentID int, sourceIP string, generation int64) (bool, error)
}
