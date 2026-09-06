package application

import (
	"context"
	"errors"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type agentLocationMapStoreStub struct {
	records []agentdomain.AgentLocationMapRecord
	err     error
	calls   int
}

func (stub *agentLocationMapStoreStub) ListLocationMapAgents(context.Context) ([]agentdomain.AgentLocationMapRecord, error) {
	stub.calls++
	return append([]agentdomain.AgentLocationMapRecord(nil), stub.records...), stub.err
}

type serverLocationMapReaderStub struct {
	location *ServerLocationRead
	err      error
	at       time.Time
	calls    int
}

func (stub *serverLocationMapReaderStub) ReadServerLocation(_ context.Context, at time.Time) (*ServerLocationRead, error) {
	stub.calls++
	stub.at = at
	return copyServerLocationRead(stub.location), stub.err
}

func TestAgentLocationMapServiceAssemblesCurrentExpiredOfflineAndActivity(t *testing.T) {
	generatedAt := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	active := 2
	zero := 0
	radius := 12.5
	store := &agentLocationMapStoreStub{records: []agentdomain.AgentLocationMapRecord{
		{
			AgentID: 1, DisplayName: "online-current", Status: "online", HealthState: "healthy", TaskSlotsUsed: &active,
			Location: agentdomain.AgentLocationSnapshot{AgentID: 1, Latitude: 1, Longitude: 2, AccuracyRadiusKM: &radius, SourceObservedIP: "8.8.8.8", ProviderKey: "freeipapi", ResolvedAt: generatedAt.Add(-7*24*time.Hour + time.Nanosecond)},
		},
		{
			AgentID: 2, DisplayName: "offline-expired", Status: "offline", HealthState: "healthy", TaskSlotsUsed: &zero,
			Location: agentdomain.AgentLocationSnapshot{AgentID: 2, Latitude: 3, Longitude: 4, SourceObservedIP: "1.1.1.1", ProviderKey: "freeipapi", ResolvedAt: generatedAt.Add(-7 * 24 * time.Hour)},
		},
		{
			AgentID: 3, DisplayName: "missing-runtime", Status: "online",
			Location: agentdomain.AgentLocationSnapshot{AgentID: 3, Latitude: 5, Longitude: 6, SourceObservedIP: "9.9.9.9", ProviderKey: "freeipapi", ResolvedAt: generatedAt, ForcedExpired: true},
		},
	}}
	serverReader := &serverLocationMapReaderStub{}
	service, err := NewAgentLocationMapService(store, serverReader, fixedClock{now: generatedAt})
	if err != nil {
		t.Fatalf("NewAgentLocationMapService: %v", err)
	}
	locationMap, err := service.Current(context.Background())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if store.calls != 1 || serverReader.calls != 1 || !serverReader.at.Equal(generatedAt) || !locationMap.GeneratedAt.Equal(generatedAt) {
		t.Fatalf("map read lifecycle store=%d server=%d at=%s generated=%s", store.calls, serverReader.calls, serverReader.at, locationMap.GeneratedAt)
	}
	if locationMap.ServerLocation != nil || len(locationMap.Agents) != 3 {
		t.Fatalf("unknown Server or complete Agent projection = %#v", locationMap)
	}
	if locationMap.Agents[0].Location.State != LocationFreshnessCurrent || locationMap.Agents[0].TaskSlotsUsed == nil || *locationMap.Agents[0].TaskSlotsUsed != 2 {
		t.Fatalf("current active Agent = %#v", locationMap.Agents[0])
	}
	if locationMap.Agents[1].Status != "offline" || locationMap.Agents[1].Location.State != LocationFreshnessExpired || locationMap.Agents[1].TaskSlotsUsed == nil || *locationMap.Agents[1].TaskSlotsUsed != 0 {
		t.Fatalf("offline expired Agent = %#v", locationMap.Agents[1])
	}
	if locationMap.Agents[2].Location.State != LocationFreshnessExpired || locationMap.Agents[2].TaskSlotsUsed != nil {
		t.Fatalf("forced-expired missing-runtime Agent = %#v", locationMap.Agents[2])
	}
}

func TestAgentLocationMapServicePropagatesMapOrServerReadFailures(t *testing.T) {
	wantErr := errors.New("unavailable")
	service, err := NewAgentLocationMapService(&agentLocationMapStoreStub{err: wantErr}, &serverLocationMapReaderStub{}, fixedClock{now: time.Now().UTC()})
	if err != nil {
		t.Fatalf("NewAgentLocationMapService: %v", err)
	}
	if _, err := service.Current(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("map store error = %v", err)
	}
	service, err = NewAgentLocationMapService(&agentLocationMapStoreStub{}, &serverLocationMapReaderStub{err: wantErr}, fixedClock{now: time.Now().UTC()})
	if err != nil {
		t.Fatalf("NewAgentLocationMapService: %v", err)
	}
	if _, err := service.Current(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Server reader error = %v", err)
	}
}

func TestNewAgentLocationMapServiceRequiresAllReadDependencies(t *testing.T) {
	clock := fixedClock{now: time.Now().UTC()}
	store := &agentLocationMapStoreStub{}
	reader := &serverLocationMapReaderStub{}
	if _, err := NewAgentLocationMapService(nil, reader, clock); err == nil {
		t.Fatal("nil map store did not fast-fail")
	}
	if _, err := NewAgentLocationMapService(store, nil, clock); err == nil {
		t.Fatal("nil Server reader did not fast-fail")
	}
	if _, err := NewAgentLocationMapService(store, reader, nil); err == nil {
		t.Fatal("nil map clock did not fast-fail")
	}
}
