package agent

import (
	"context"
	"time"

	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	systemapp "github.com/yyhuni/lunafox/server/internal/modules/system/application"
)

type agentServerLocationReaderAdapter struct {
	reader *systemapp.ServerLocationReader
}

func (adapter *agentServerLocationReaderAdapter) ReadServerLocation(ctx context.Context, at time.Time) (*agentapp.ServerLocationRead, error) {
	projection, err := adapter.reader.ReadAt(ctx, at)
	if err != nil || projection == nil {
		return nil, err
	}
	return &agentapp.ServerLocationRead{
		State:            agentapp.LocationFreshness(projection.State),
		ObservedEgressIP: projection.ObservedEgressIP,
		Latitude:         projection.Latitude,
		Longitude:        projection.Longitude,
		AccuracyRadiusKM: copyAgentServerLocationRadius(projection.AccuracyRadiusKM),
		ProviderKey:      projection.ProviderKey,
		ResolvedAt:       projection.ResolvedAt.UTC(),
	}, nil
}

func copyAgentServerLocationRadius(radius *float64) *float64 {
	if radius == nil {
		return nil
	}
	copy := *radius
	return &copy
}
