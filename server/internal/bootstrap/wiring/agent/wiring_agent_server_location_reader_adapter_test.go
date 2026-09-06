package agent

import (
	"context"
	"testing"
	"time"

	systemapp "github.com/yyhuni/lunafox/server/internal/modules/system/application"
	systemdomain "github.com/yyhuni/lunafox/server/internal/modules/system/domain"
)

type agentServerLocationRepositoryStub struct {
	snapshot *systemdomain.ServerLocationSnapshot
}

func (stub *agentServerLocationRepositoryStub) Get(context.Context) (*systemdomain.ServerLocationSnapshot, error) {
	return stub.snapshot, nil
}

func (stub *agentServerLocationRepositoryStub) Replace(context.Context, systemdomain.ServerLocationSnapshot) error {
	return nil
}

func (stub *agentServerLocationRepositoryStub) MarkExpired(context.Context) error { return nil }

type agentServerLocationClockStub struct{ now time.Time }

func (stub agentServerLocationClockStub) NowUTC() time.Time { return stub.now }

func TestAgentServerLocationReaderAdapterMapsExactExpiryAndUnknown(t *testing.T) {
	resolvedAt := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	repository := &agentServerLocationRepositoryStub{snapshot: &systemdomain.ServerLocationSnapshot{
		ObservedEgressIP: "8.8.8.8",
		Latitude:         1,
		Longitude:        2,
		ProviderKey:      "freeipapi",
		ResolvedAt:       resolvedAt,
		UpdatedAt:        resolvedAt,
	}}
	reader, err := systemapp.NewServerLocationReader(repository, agentServerLocationClockStub{now: resolvedAt})
	if err != nil {
		t.Fatalf("NewServerLocationReader: %v", err)
	}
	adapter, err := NewAgentServerLocationReaderAdapter(reader)
	if err != nil {
		t.Fatalf("NewAgentServerLocationReaderAdapter: %v", err)
	}
	projection, err := adapter.ReadServerLocation(context.Background(), resolvedAt.Add(7*24*time.Hour))
	if err != nil || projection == nil || projection.State != "expired" || projection.ObservedEgressIP != "8.8.8.8" {
		t.Fatalf("expired projection = (%#v, %v)", projection, err)
	}

	repository.snapshot = nil
	projection, err = adapter.ReadServerLocation(context.Background(), resolvedAt)
	if err != nil || projection != nil {
		t.Fatalf("unknown projection = (%#v, %v)", projection, err)
	}
}

func TestNewAgentServerLocationReaderAdapterRequiresSystemReader(t *testing.T) {
	if _, err := NewAgentServerLocationReaderAdapter(nil); err == nil {
		t.Fatal("nil system reader did not fast-fail")
	}
}
