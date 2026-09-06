package domain

import (
	"context"
	"time"
)

const ServerLocationFreshness = 7 * 24 * time.Hour

// ServerLocationSnapshot is the complete last-successful outbound egress location.
type ServerLocationSnapshot struct {
	ObservedEgressIP string
	Latitude         float64
	Longitude        float64
	AccuracyRadiusKM *float64
	ProviderKey      string
	ResolvedAt       time.Time
	ForcedExpired    bool
	UpdatedAt        time.Time
}

func (snapshot ServerLocationSnapshot) IsExpiredAt(now time.Time) bool {
	return snapshot.ForcedExpired || !now.Before(snapshot.ResolvedAt.Add(ServerLocationFreshness))
}

type ServerLocationRepository interface {
	Get(ctx context.Context) (*ServerLocationSnapshot, error)
	Replace(ctx context.Context, snapshot ServerLocationSnapshot) error
	MarkExpired(ctx context.Context) error
}
