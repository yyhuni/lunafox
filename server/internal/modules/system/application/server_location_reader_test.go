package application

import (
	"context"
	"testing"
	"time"

	systemdomain "github.com/yyhuni/lunafox/server/internal/modules/system/domain"
)

func TestServerLocationReaderUsesExactSevenDayFreshnessBoundary(t *testing.T) {
	resolvedAt := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	repository := &serverLocationRepositoryStub{snapshot: &systemdomain.ServerLocationSnapshot{
		ObservedEgressIP: "8.8.8.8",
		Latitude:         1,
		Longitude:        2,
		ProviderKey:      "freeipapi",
		ResolvedAt:       resolvedAt,
		UpdatedAt:        resolvedAt,
	}}
	clock := &serverLocationClockStub{now: resolvedAt.Add(7*24*time.Hour - time.Nanosecond)}
	reader, err := NewServerLocationReader(repository, clock)
	if err != nil {
		t.Fatalf("NewServerLocationReader: %v", err)
	}
	projection, err := reader.Read(context.Background())
	if err != nil || projection == nil || projection.State != ServerLocationStateCurrent {
		t.Fatalf("projection before expiry = (%#v, %v)", projection, err)
	}
	clock.now = resolvedAt.Add(7 * 24 * time.Hour)
	projection, err = reader.Read(context.Background())
	if err != nil || projection == nil || projection.State != ServerLocationStateExpired || projection.ObservedEgressIP != "8.8.8.8" || !projection.ResolvedAt.Equal(resolvedAt) {
		t.Fatalf("projection at expiry = (%#v, %v)", projection, err)
	}
}

func TestServerLocationReaderReturnsUnknownWithoutSuccess(t *testing.T) {
	reader, err := NewServerLocationReader(&serverLocationRepositoryStub{}, &serverLocationClockStub{now: time.Now().UTC()})
	if err != nil {
		t.Fatalf("NewServerLocationReader: %v", err)
	}
	projection, err := reader.Read(context.Background())
	if err != nil || projection != nil {
		t.Fatalf("unknown projection = (%#v, %v), want nil", projection, err)
	}
}

type serverLocationRepositoryStub struct {
	snapshot *systemdomain.ServerLocationSnapshot
	err      error
}

func (repository *serverLocationRepositoryStub) Get(context.Context) (*systemdomain.ServerLocationSnapshot, error) {
	return repository.snapshot, repository.err
}

func (repository *serverLocationRepositoryStub) Replace(context.Context, systemdomain.ServerLocationSnapshot) error {
	return repository.err
}

func (repository *serverLocationRepositoryStub) MarkExpired(context.Context) error {
	return repository.err
}

type serverLocationClockStub struct{ now time.Time }

func (clock *serverLocationClockStub) NowUTC() time.Time { return clock.now }
