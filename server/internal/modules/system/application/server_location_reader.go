package application

import (
	"context"
	"errors"
	"time"

	systemdomain "github.com/yyhuni/lunafox/server/internal/modules/system/domain"
)

type ServerLocationState string

const (
	ServerLocationStateCurrent ServerLocationState = "current"
	ServerLocationStateExpired ServerLocationState = "expired"
)

// ServerLocationProjection is a provider-neutral complete last-successful snapshot and freshness state.
type ServerLocationProjection struct {
	State            ServerLocationState
	ObservedEgressIP string
	Latitude         float64
	Longitude        float64
	AccuracyRadiusKM *float64
	ProviderKey      string
	ResolvedAt       time.Time
}

type ServerLocationClock interface {
	NowUTC() time.Time
}

// ServerLocationReader restores the singleton success without exposing persistence types.
type ServerLocationReader struct {
	repository systemdomain.ServerLocationRepository
	clock      ServerLocationClock
}

func NewServerLocationReader(repository systemdomain.ServerLocationRepository, clock ServerLocationClock) (*ServerLocationReader, error) {
	if repository == nil {
		return nil, errors.New("Server location repository is required")
	}
	if clock == nil {
		return nil, errors.New("Server location clock is required")
	}
	return &ServerLocationReader{repository: repository, clock: clock}, nil
}

func (reader *ServerLocationReader) Read(ctx context.Context) (*ServerLocationProjection, error) {
	return reader.ReadAt(ctx, reader.clock.NowUTC())
}

func (reader *ServerLocationReader) ReadAt(ctx context.Context, at time.Time) (*ServerLocationProjection, error) {
	snapshot, err := reader.repository.Get(ctx)
	if err != nil || snapshot == nil {
		return nil, err
	}
	state := ServerLocationStateCurrent
	if snapshot.IsExpiredAt(at.UTC()) {
		state = ServerLocationStateExpired
	}
	return &ServerLocationProjection{
		State:            state,
		ObservedEgressIP: snapshot.ObservedEgressIP,
		Latitude:         snapshot.Latitude,
		Longitude:        snapshot.Longitude,
		AccuracyRadiusKM: copyServerLocationProjectionRadius(snapshot.AccuracyRadiusKM),
		ProviderKey:      snapshot.ProviderKey,
		ResolvedAt:       snapshot.ResolvedAt.UTC(),
	}, nil
}

func copyServerLocationProjectionRadius(radius *float64) *float64 {
	if radius == nil {
		return nil
	}
	copy := *radius
	return &copy
}
