package main

import (
	"context"
	"testing"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

func TestSmokeAuthorityObservesReconnectWithoutSessionDrift(t *testing.T) {
	authority := newSmokeAuthority("cut-smoke-agent-token-reconnect", nil)
	heartbeat := agentdomain.AgentHeartbeatEvent{
		SessionID: "stable-session", SessionEpoch: 7,
		OperatingSystem: "linux", Architecture: "amd64", ContainerRuntimeReady: true,
		SupportedEngineAPIMajors: []uint32{2},
	}
	if err := authority.OnConnected(context.Background(), authority.agent, "192.0.2.10"); err != nil {
		t.Fatalf("first OnConnected: %v", err)
	}
	if authority.agent.ObservedSourceIP != "192.0.2.10" || authority.agent.ObservedIPGeneration != 1 {
		t.Fatalf("first connection observation = %q generation %d", authority.agent.ObservedSourceIP, authority.agent.ObservedIPGeneration)
	}
	if err := authority.RecordHeartbeat(context.Background(), smokeAgentID, heartbeat); err != nil {
		t.Fatalf("first RecordHeartbeat: %v", err)
	}
	if err := authority.OnDisconnected(context.Background(), smokeAgentID); err != nil {
		t.Fatalf("OnDisconnected: %v", err)
	}
	if err := authority.OnConnected(context.Background(), authority.agent, "192.0.2.10"); err != nil {
		t.Fatalf("second OnConnected: %v", err)
	}
	if authority.agent.ObservedIPGeneration != 1 {
		t.Fatalf("same-source reconnect generation = %d, want 1", authority.agent.ObservedIPGeneration)
	}
	if err := authority.RecordHeartbeat(context.Background(), smokeAgentID, heartbeat); err != nil {
		t.Fatalf("second RecordHeartbeat: %v", err)
	}
	connections, passed := authority.sameSessionReconnectEvidence()
	if connections != 2 || !passed {
		t.Fatalf("same-session reconnect evidence = connections %d passed %t", connections, passed)
	}

	heartbeat.SessionID = "replacement-session"
	heartbeat.SessionEpoch++
	if err := authority.RecordHeartbeat(context.Background(), smokeAgentID, heartbeat); err != nil {
		t.Fatalf("replacement RecordHeartbeat: %v", err)
	}
	if _, passed := authority.sameSessionReconnectEvidence(); passed {
		t.Fatal("session identity drift satisfied same-session reconnect evidence")
	}
}

func TestRunSmokeSessionFenceEvidenceUsesProductionRepository(t *testing.T) {
	packages := smokePackageReaderForRepositoryTest(t)
	wordlists, err := newSmokeWordlists()
	if err != nil {
		t.Fatalf("newSmokeWordlists: %v", err)
	}
	defer wordlists.cleanup()
	compiler, err := scanapp.NewPlanTaskCompiler(packages, wordlists)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler: %v", err)
	}

	evidence, err := runSmokeSessionFenceEvidence(context.Background(), compiler, packages)
	if err != nil {
		t.Fatalf("runSmokeSessionFenceEvidence: %v", err)
	}
	if !evidence.Passed || evidence.TaskStatus != "failed" || evidence.ScanStatus != "failed" ||
		evidence.FailureKind != "agent_disconnected" || !evidence.StaleTerminalRejected || !evidence.ReplayPassed {
		t.Fatalf("session-fence evidence = %#v", evidence)
	}
}
