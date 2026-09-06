package agentcontrol

import (
	"context"
	"testing"
	"time"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"google.golang.org/grpc/metadata"
)

func TestGeoIPProviderFailureCannotChangeReadyHeartbeatOrClaimBehavior(t *testing.T) {
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	locationRepository := &controlLocationRepositoryStub{expired: make(chan controlLocationExpiration, 1)}
	observer, err := agentapp.NewAgentLocationObserver(
		locationRepository,
		controlProviderFailureLookup{},
		controlLocationClock{now: base},
	)
	if err != nil {
		t.Fatalf("NewAgentLocationObserver: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := observer.Shutdown(ctx); err != nil {
			t.Errorf("observer shutdown: %v", err)
		}
	})

	agent := &agentdomain.Agent{
		ID:                   77,
		InstanceID:           "agt-77",
		ObservedSourceIP:     "8.8.8.8",
		ObservedIPGeneration: 2,
		Location: &agentdomain.AgentLocationSnapshot{
			AgentID:          77,
			SourceObservedIP: "1.1.1.1",
			ResolvedAt:       base.Add(-time.Hour),
		},
	}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	tasks := &taskRuntimeStub{}
	service := NewControlPlaneService(finder, lifecycle, tasks).WithReadyConnectionObserver(observer)
	streamContext := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-77"))
	stream := &fakeConnectStream{
		ctx: streamContext,
		incoming: []*agentcontrolv1.ConnectRequest{
			runtimeRegisterSessionRequest("agt-77", "session-77"),
			{Payload: &agentcontrolv1.ConnectRequest_RequestTask{RequestTask: &agentcontrolv1.RequestTask{
				RequestId: "550e8400-e29b-41d4-a716-446655440000",
			}}},
			{Payload: &agentcontrolv1.ConnectRequest_Heartbeat{Heartbeat: &agentcontrolv1.Heartbeat{
				Agent:                 resourcenames.Agent("agt-77"),
				Session:               resourcenames.AgentSession("agt-77", "session-77"),
				ObservedHostname:      "node-77",
				CompatibilityRevision: testCompatibilityRevision,
			}}},
		},
	}

	if err := service.Connect(stream); err != nil {
		t.Fatalf("Connect changed by GeoIP failure: %v", err)
	}
	if finder.lastAuthenticationToken != "agent-token-77" || lifecycle.connectedCount != 1 || len(lifecycle.heartbeats) != 2 || tasks.claimCalls != 1 {
		t.Fatalf("control path after GeoIP failure: auth=%q connected=%d heartbeats=%d claims=%d", finder.lastAuthenticationToken, lifecycle.connectedCount, len(lifecycle.heartbeats), tasks.claimCalls)
	}
	if len(stream.sent) != 2 || stream.sent[0].GetSessionReady() == nil || stream.sent[1].GetTaskAssign() == nil {
		t.Fatalf("control responses after GeoIP failure = %#v", stream.sent)
	}
	select {
	case expiration := <-locationRepository.expired:
		if expiration.agentID != 77 || expiration.sourceIP != "8.8.8.8" || expiration.generation != 2 {
			t.Fatalf("failure expiration = %#v", expiration)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for isolated GeoIP failure callback")
	}
}

type controlLocationClock struct{ now time.Time }

func (clock controlLocationClock) NowUTC() time.Time { return clock.now }

type controlProviderFailureLookup struct{}

func (controlProviderFailureLookup) SubmitIP(_ context.Context, _ string, completion func(agentapp.AgentLocationLookupOutcome)) error {
	go completion(agentapp.AgentLocationLookupOutcome{
		FailureClass:      "network",
		ProviderAttempted: true,
		CompletedAt:       time.Now().UTC(),
	})
	return nil
}

type controlLocationRepositoryStub struct {
	expired chan controlLocationExpiration
}

type controlLocationExpiration struct {
	agentID    int
	sourceIP   string
	generation int64
}

func (*controlLocationRepositoryStub) GetLocation(context.Context, int) (*agentdomain.AgentLocationSnapshot, error) {
	return nil, nil
}

func (*controlLocationRepositoryStub) ReplaceLocationIfObservationMatches(context.Context, agentdomain.AgentLocationSnapshot, int64) (bool, error) {
	return false, nil
}

func (repository *controlLocationRepositoryStub) MarkLocationExpiredIfObservationMatches(_ context.Context, agentID int, sourceIP string, generation int64) (bool, error) {
	repository.expired <- controlLocationExpiration{agentID: agentID, sourceIP: sourceIP, generation: generation}
	return true, nil
}
