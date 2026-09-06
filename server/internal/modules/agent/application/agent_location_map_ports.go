package application

import (
	"context"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type AgentLocationMapStore interface {
	ListLocationMapAgents(ctx context.Context) ([]agentdomain.AgentLocationMapRecord, error)
}

type ServerLocationRead struct {
	State            LocationFreshness
	ObservedEgressIP string
	Latitude         float64
	Longitude        float64
	AccuracyRadiusKM *float64
	ProviderKey      string
	ResolvedAt       time.Time
}

type ServerLocationReader interface {
	ReadServerLocation(ctx context.Context, at time.Time) (*ServerLocationRead, error)
}
