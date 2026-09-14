package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	upgradeapp "github.com/yyhuni/lunafox/server/internal/modules/upgrade/application"
	upgradedomain "github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

type upgradeAgentRepositoryStub struct {
	agentdomain.AgentOperationalRepository
	agents []*agentdomain.Agent
}

func (stub *upgradeAgentRepositoryStub) List(_ context.Context, _ int, _ int, _ string, _ string) ([]*agentdomain.Agent, int64, error) {
	return stub.agents, int64(len(stub.agents)), nil
}

type upgradeAgentPublisherStub struct {
	agentapp.AgentMessagePublisher
	calls []struct {
		id      int
		payload agentdomain.UpdateRequiredPayload
	}
}

func (stub *upgradeAgentPublisherStub) SendUpdateRequired(id int, payload agentdomain.UpdateRequiredPayload) bool {
	stub.calls = append(stub.calls, struct {
		id      int
		payload agentdomain.UpdateRequiredPayload
	}{id: id, payload: payload})
	return true
}

func TestUpgradeAgentSourceSnapshotsReadinessWithoutSecrets(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	heartbeat := now.Add(-30 * time.Second)
	agent := &agentdomain.Agent{ID: 7, Status: "online", HealthState: "healthy", AgentVersion: "1.2.3", OperatingSystem: "linux", Architecture: "amd64", ContainerRuntimeReady: true, SupportedEngineAPIMajors: []uint32{5}, SessionID: "session-7", SessionEpoch: 4, LastHeartbeat: &heartbeat, AuthenticationToken: "must-not-be-copied"}
	source := newUpgradeAgentSource(&upgradeAgentRepositoryStub{agents: []*agentdomain.Agent{agent}}, nil, func() time.Time { return now })
	expectations, err := source.Snapshot(context.Background(), upgradeapp.AgentUpgradeTarget{Version: "1.2.3", Digest: "sha256:" + strings.Repeat("a", 64), ImageRef: "registry.example/agent@sha256:" + strings.Repeat("a", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if len(expectations) != 1 || !expectations[0].Ready() {
		t.Fatalf("expectations=%#v", expectations)
	}
	if expectations[0].ObservedDigest != "" || expectations[0].Diagnostic != "" {
		t.Fatalf("fabricated or unexpected evidence=%#v", expectations[0])
	}
}

func TestUpgradeAgentSourceReconcilesOnlyExpectedAgentsAndPublishesTarget(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	heartbeat := now.Add(-10 * time.Second)
	agent := &agentdomain.Agent{ID: 9, Status: "online", HealthState: "healthy", AgentVersion: "2.0.0", OperatingSystem: "linux", Architecture: "arm64", ContainerRuntimeReady: true, SupportedEngineAPIMajors: []uint32{5}, SessionID: "s9", SessionEpoch: 2, LastHeartbeat: &heartbeat}
	repository := &upgradeAgentRepositoryStub{agents: []*agentdomain.Agent{agent}}
	publisher := &upgradeAgentPublisherStub{}
	source := newUpgradeAgentSource(repository, publisher, func() time.Time { return now })
	target := upgradeapp.AgentUpgradeTarget{Version: "2.0.0", Digest: "sha256:" + strings.Repeat("b", 64), ImageRef: "registry.example/agent@sha256:" + strings.Repeat("b", 64)}
	expected := []upgradedomain.AgentExpectation{{AgentID: 9, DesiredVersion: target.Version, TargetDigest: target.Digest}, {AgentID: 10, DesiredVersion: target.Version, TargetDigest: target.Digest}}
	refreshed, err := source.Reconcile(context.Background(), target, expected)
	if err != nil {
		t.Fatal(err)
	}
	if len(refreshed) != 2 || refreshed[0].AgentID != 9 || refreshed[1].AgentID != 10 || refreshed[1].Connected || refreshed[1].ClaimReady {
		t.Fatalf("reconcile=%#v", refreshed)
	}
	if err := source.NotifyUpdateRequired(context.Background(), target, expected); err != nil {
		t.Fatal(err)
	}
	if len(publisher.calls) != 2 || publisher.calls[0].payload.AgentVersion != target.Version || publisher.calls[0].payload.AgentImageRef != target.ImageRef {
		t.Fatalf("publisher calls=%#v", publisher.calls)
	}
}

func TestUpgradeAgentSourceClassifiesStaleAndPausedAgents(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	stale := now.Add(-2 * time.Minute)
	pausedHeartbeat := now.Add(-5 * time.Second)
	agents := []*agentdomain.Agent{
		{ID: 1, Status: "online", HealthState: "healthy", AgentVersion: "1.0.0", LastHeartbeat: &stale},
		{ID: 2, Status: "online", HealthState: "paused", AgentVersion: "1.0.0", LastHeartbeat: &pausedHeartbeat},
	}
	source := newUpgradeAgentSource(&upgradeAgentRepositoryStub{agents: agents}, nil, func() time.Time { return now })
	got, err := source.Snapshot(context.Background(), upgradeapp.AgentUpgradeTarget{Version: "1.0.0", Digest: "sha256:" + strings.Repeat("c", 64), ImageRef: "agent@sha256:" + strings.Repeat("c", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Connected || !got[1].Paused || got[1].Healthy {
		t.Fatalf("classification=%#v", got)
	}
}
