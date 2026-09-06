package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type agentClusterSummaryStoreStub struct {
	aggregate   agentdomain.AgentClusterAggregate
	err         error
	generatedAt time.Time
	calls       int
}

func (stub *agentClusterSummaryStoreStub) GetClusterAggregate(_ context.Context, generatedAt time.Time) (agentdomain.AgentClusterAggregate, error) {
	stub.calls++
	stub.generatedAt = generatedAt
	return stub.aggregate, stub.err
}

func TestAgentClusterSummaryServiceUsesOneClockInstantAndMapsProjection(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 123, time.FixedZone("test", 8*60*60))
	store := &agentClusterSummaryStoreStub{aggregate: agentdomain.AgentClusterAggregate{
		TotalNodes:         3,
		HealthyCount:       2,
		OfflineCount:       1,
		StaleAgentCount:    1,
		ConfiguredSlots:    10,
		OccupiedSlots:      4,
		AvailableSlots:     3,
		UnavailableSlots:   3,
		OvercommittedSlots: 2,
		PositionedCount:    2,
		UnpositionedCount:  1,
	}}
	service, err := NewAgentClusterSummaryService(store, fixedClock{now: now})
	if err != nil {
		t.Fatalf("NewAgentClusterSummaryService: %v", err)
	}
	summary, err := service.Current(context.Background())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	wantGeneratedAt := now.UTC()
	if store.calls != 1 || !store.generatedAt.Equal(wantGeneratedAt) || !summary.GeneratedAt.Equal(wantGeneratedAt) {
		t.Fatalf("generation instant store=%s summary=%s calls=%d", store.generatedAt, summary.GeneratedAt, store.calls)
	}
	if summary.ExecutionFreshnessSeconds != 15 || summary.Nodes.Total != 3 || summary.Nodes.Healthy != 2 || summary.Nodes.Offline != 1 || summary.Nodes.Stale != 1 {
		t.Fatalf("summary nodes = %#v", summary)
	}
	if summary.ExecutionCapacity.Configured != 10 || summary.ExecutionCapacity.Occupied != 4 || summary.ExecutionCapacity.Available != 3 || summary.ExecutionCapacity.Unavailable != 3 || summary.ExecutionCapacity.Overcommitted != 2 {
		t.Fatalf("summary capacity = %#v", summary.ExecutionCapacity)
	}
	if summary.State != AgentClusterStateNeedsAttention || !reflect.DeepEqual(summary.ReasonCodes, []AgentClusterReasonCode{
		AgentClusterReasonOfflineAgents,
		AgentClusterReasonStaleRuntimeObservations,
		AgentClusterReasonOvercommittedSlots,
	}) {
		t.Fatalf("summary conclusion = state=%s reasons=%v", summary.State, summary.ReasonCodes)
	}
	if summary.LocationCoverage.Positioned != 2 || summary.LocationCoverage.Unpositioned != 1 {
		t.Fatalf("summary coverage = %#v", summary.LocationCoverage)
	}
}

func TestAgentClusterSummaryServiceReturnsStoreFailure(t *testing.T) {
	wantErr := errors.New("aggregate unavailable")
	service, err := NewAgentClusterSummaryService(&agentClusterSummaryStoreStub{err: wantErr}, fixedClock{now: time.Now().UTC()})
	if err != nil {
		t.Fatalf("NewAgentClusterSummaryService: %v", err)
	}
	if _, err := service.Current(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Current error = %v, want %v", err, wantErr)
	}
}

func TestNewAgentClusterSummaryServiceRequiresReadOnlyDependencies(t *testing.T) {
	if _, err := NewAgentClusterSummaryService(nil, fixedClock{now: time.Now().UTC()}); err == nil {
		t.Fatal("nil summary store did not fast-fail")
	}
	if _, err := NewAgentClusterSummaryService(&agentClusterSummaryStoreStub{}, nil); err == nil {
		t.Fatal("nil summary clock did not fast-fail")
	}
}
