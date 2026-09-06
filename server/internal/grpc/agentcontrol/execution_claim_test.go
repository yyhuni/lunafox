package agentcontrol

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type executionClaimRuntimeStub struct {
	taskRuntimeStub
	plan         *agentexecutionv1.ResolvedEngineExecutionPlan
	claimErr     error
	calls        int
	agentID      int
	sessionID    string
	sessionEpoch int64
	requestID    string
	snapshot     agentdomain.AgentExecutionCapabilitySnapshot
	claimHook    func()
}

func (stub *executionClaimRuntimeStub) ClaimNextExecutionPlan(_ context.Context, agentID int, sessionID string, sessionEpoch int64, requestID string, snapshot agentdomain.AgentExecutionCapabilitySnapshot) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	stub.calls++
	stub.agentID = agentID
	stub.sessionID = sessionID
	stub.sessionEpoch = sessionEpoch
	stub.requestID = requestID
	stub.snapshot = snapshot.Clone()
	if stub.claimHook != nil {
		stub.claimHook()
	}
	return stub.plan, stub.claimErr
}

func TestConnectRequestTaskSendsCorrelatedSavedPlanOrExplicitNoTask(t *testing.T) {
	for _, test := range []struct {
		name string
		plan *agentexecutionv1.ResolvedEngineExecutionPlan
	}{
		{name: "saved plan", plan: &agentexecutionv1.ResolvedEngineExecutionPlan{Execution: "executions/7"}},
		{name: "scheduler miss"},
	} {
		t.Run(test.name, func(t *testing.T) {
			agent := &agentdomain.Agent{ID: 72, InstanceID: "agt-72"}
			claims := &executionClaimRuntimeStub{plan: test.plan}
			service := NewControlPlaneService(&agentFinderStub{agent: agent}, &runtimeLifecycleStub{}, claims)
			requestID := "550e8400-e29b-41d4-a716-446655440000"
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-72"))
			stream := &fakeConnectStream{ctx: ctx, incoming: []*agentcontrolv1.ConnectRequest{
				runtimeRegisterSessionRequest("agt-72", "session-72"),
				{Payload: &agentcontrolv1.ConnectRequest_RequestTask{RequestTask: &agentcontrolv1.RequestTask{RequestId: requestID}}},
			}}
			if err := service.Connect(stream); err != nil {
				t.Fatalf("Connect() error = %v", err)
			}
			if claims.calls != 1 || claims.agentID != agent.ID || claims.sessionID != "session-72" || claims.sessionEpoch <= 0 || claims.requestID != requestID || !claims.snapshot.ContainerRuntimeReady || len(claims.snapshot.SupportedEngineAPIMajors) != 1 || claims.snapshot.SupportedEngineAPIMajors[0] != 2 {
				t.Fatalf("claim arguments = %#v", claims)
			}
			if len(stream.sent) != 2 {
				t.Fatalf("sent events = %d, want SessionReady and TaskAssign", len(stream.sent))
			}
			assignment := stream.sent[1].GetTaskAssign()
			if assignment == nil || assignment.GetRequestId() != requestID {
				t.Fatalf("TaskAssign = %#v", assignment)
			}
			if test.plan == nil {
				if assignment.GetNoTask() == nil || assignment.GetPlan() != nil {
					t.Fatalf("scheduler miss outcome = %#v", assignment.GetOutcome())
				}
			} else if assignment.GetPlan() != test.plan || assignment.GetNoTask() != nil {
				t.Fatalf("saved-plan outcome = %#v", assignment.GetOutcome())
			}
		})
	}
}

func TestConnectRequestTaskFailsClosedWithoutUsableExecutionSnapshot(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	previousLogger := zap.L()
	zap.ReplaceGlobals(zap.New(core))
	t.Cleanup(func() { zap.ReplaceGlobals(previousLogger) })
	agent := &agentdomain.Agent{ID: 73, InstanceID: "agt-73"}
	claims := &executionClaimRuntimeStub{plan: &agentexecutionv1.ResolvedEngineExecutionPlan{Execution: "must-not-claim"}}
	service := NewControlPlaneService(&agentFinderStub{agent: agent}, &runtimeLifecycleStub{}, claims)
	registration := runtimeRegisterSessionRequest("agt-73", "session-73")
	registration.GetRegisterSession().OperatingSystem = ""
	registration.GetRegisterSession().Architecture = ""
	registration.GetRegisterSession().ContainerRuntimeReady = false
	registration.GetRegisterSession().SupportedEngineApiMajors = nil
	requestID := "550e8400-e29b-41d4-a716-446655440000"
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-73"))
	stream := &fakeConnectStream{ctx: ctx, incoming: []*agentcontrolv1.ConnectRequest{
		registration,
		{Payload: &agentcontrolv1.ConnectRequest_RequestTask{RequestTask: &agentcontrolv1.RequestTask{RequestId: requestID}}},
	}}
	if err := service.Connect(stream); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if claims.calls != 0 {
		t.Fatalf("claim called %d times without usable snapshot", claims.calls)
	}
	assignment := stream.sent[1].GetTaskAssign()
	if assignment == nil || assignment.GetRequestId() != requestID || assignment.GetNoTask() == nil {
		t.Fatalf("fail-closed assignment = %#v", assignment)
	}
	entries := logs.FilterMessage("engine_execution_claim_deferred").All()
	if len(entries) != 1 || entries[0].ContextMap()["error.reason"] != string(executionSnapshotAdmissionRuntimeNotReady) || entries[0].ContextMap()["execution.phase"] != "scheduler_admission" {
		t.Fatalf("bounded claim diagnostic = %#v", entries)
	}
	if _, exposed := entries[0].ContextMap()["snapshot"]; exposed {
		t.Fatal("claim diagnostic exposed capability snapshot")
	}
}

func TestExecutionCapabilityAllowsUnavailableDaemonWithoutPlatformFacts(t *testing.T) {
	snapshot, err := validatedExecutionCapabilitySnapshot("v1", "", "", false, []uint32{2}, 0, 0)
	if err != nil {
		t.Fatalf("unavailable daemon snapshot error = %v", err)
	}
	if snapshot.ContainerRuntimeReady || snapshot.OperatingSystem != "" || snapshot.Architecture != "" {
		t.Fatalf("unavailable daemon snapshot = %#v", snapshot)
	}
	if _, err := validatedExecutionCapabilitySnapshot("v1", "", "", true, []uint32{2}, 0, 0); err == nil {
		t.Fatal("ready daemon without platform facts was accepted")
	}
}

func TestConnectRequestTaskDoesNotSendAssignmentAfterSessionTakeover(t *testing.T) {
	agent := &agentdomain.Agent{ID: 74, InstanceID: "agt-74"}
	claims := &executionClaimRuntimeStub{plan: &agentexecutionv1.ResolvedEngineExecutionPlan{Execution: "executions/7"}}
	service := NewControlPlaneService(&agentFinderStub{agent: agent}, &runtimeLifecycleStub{}, claims)
	registry := NewActiveSessionRegistry()
	service.WithActiveSessionRegistry(registry)
	claims.claimHook = func() {
		registry.Register(agent.ID, 999, "replacement-session")
	}
	requestID := "550e8400-e29b-41d4-a716-446655440000"
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-74"))
	stream := &fakeConnectStream{ctx: ctx, incoming: []*agentcontrolv1.ConnectRequest{
		runtimeRegisterSessionRequest("agt-74", "session-74"),
		{Payload: &agentcontrolv1.ConnectRequest_RequestTask{RequestTask: &agentcontrolv1.RequestTask{RequestId: requestID}}},
	}}
	err := service.Connect(stream)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Connect() after claim-time takeover error = %v", err)
	}
	if len(stream.sent) != 1 || stream.sent[0].GetSessionReady() == nil {
		t.Fatalf("stale stream sent assignment after takeover: %#v", stream.sent)
	}
}

func TestTaskAssignNoTaskDoesNotSendAfterSessionTakeover(t *testing.T) {
	agent := &agentdomain.Agent{ID: 75, InstanceID: "agt-75"}
	service := NewControlPlaneService(&agentFinderStub{agent: agent}, &runtimeLifecycleStub{}, &executionClaimRuntimeStub{})
	registry := NewActiveSessionRegistry()
	service.WithActiveSessionRegistry(registry)
	stale, _ := registry.Register(agent.ID, 750, "session-75")
	stale, ok := registry.MarkReadyIfCurrent(agent.ID, stale.SessionID, stale.SessionEpoch, stale.StreamID)
	if !ok {
		t.Fatal("mark stale fixture ready")
	}
	registry.Register(agent.ID, 751, "replacement-session")
	stream := &fakeConnectStream{ctx: context.Background()}
	err := service.sendTaskAssignIfCurrent(agent.ID, stale, &sync.Mutex{}, stream, "550e8400-e29b-41d4-a716-446655440000", nil)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("no_task send after takeover error = %v", err)
	}
	if len(stream.sent) != 0 {
		t.Fatalf("stale stream received no_task after takeover: %#v", stream.sent)
	}
}

func TestHeartbeatPersistenceFailureDoesNotExtendInMemoryLease(t *testing.T) {
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	registry := newActiveSessionRegistry(DefaultSessionHeartbeatTimeout, func() time.Time { return now })
	agent := &agentdomain.Agent{ID: 76, InstanceID: "agt-76"}
	session, _ := registry.Register(agent.ID, 760, "session-76")
	session, ok := registry.MarkReadyIfCurrent(agent.ID, session.SessionID, session.SessionEpoch, session.StreamID)
	if !ok {
		t.Fatal("mark heartbeat fixture ready")
	}
	originalLastSeen := registry.sessions[agent.ID].LastSeenAt
	service := NewControlPlaneService(
		&agentFinderStub{agent: agent},
		&runtimeLifecycleStub{handleErr: errors.New("database unavailable")},
		&executionClaimRuntimeStub{},
	)
	service.WithActiveSessionRegistry(registry)
	now = now.Add(30 * time.Second)
	err := service.handleHeartbeat(context.Background(), agent, session, &agentcontrolv1.Heartbeat{
		Agent:                    resourcenames.Agent(agent.InstanceID),
		Session:                  resourcenames.AgentSession(agent.InstanceID, session.SessionID),
		ObservedHostname:         "node-76",
		AgentVersion:             "v1.0.0",
		OperatingSystem:          "linux",
		Architecture:             "amd64",
		ContainerRuntimeReady:    true,
		SupportedEngineApiMajors: []uint32{2},
		CompatibilityRevision:    testCompatibilityRevision,
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("heartbeat persistence failure = %v", err)
	}
	current := registry.sessions[agent.ID]
	if !current.LastSeenAt.Equal(originalLastSeen) {
		t.Fatalf("failed heartbeat extended in-memory lease from %s to %s", originalLastSeen, current.LastSeenAt)
	}
	if current.ExecutionSnapshot != nil {
		t.Fatalf("failed heartbeat updated capability snapshot: %#v", current.ExecutionSnapshot)
	}
}

func TestConnectTerminalResultDoesNotAckAfterSessionTakeover(t *testing.T) {
	agent := &agentdomain.Agent{ID: 77, InstanceID: "agt-77"}
	tasks := &taskRuntimeStub{}
	service := NewControlPlaneService(&agentFinderStub{agent: agent}, &runtimeLifecycleStub{}, tasks)
	registry := NewActiveSessionRegistry()
	service.WithActiveSessionRegistry(registry)
	tasks.reportHook = func() {
		registry.Register(agent.ID, 999, "replacement-session")
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-77"))
	stream := &fakeConnectStream{ctx: ctx, incoming: []*agentcontrolv1.ConnectRequest{
		runtimeRegisterSessionRequest(agent.InstanceID, "session-77"),
		{Payload: &agentcontrolv1.ConnectRequest_TerminalTaskResult{TerminalTaskResult: &agentcontrolv1.TerminalTaskResult{
			Task: resourcenames.Task(70, 701), Result: agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_SUCCEEDED, CompatibilityRevision: testCompatibilityRevision,
			Diagnostics: terminalDiagnosticsProtoForTest(),
		}}},
	}}
	err := service.Connect(stream)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Connect() after terminal-time takeover error = %v", err)
	}
	if len(tasks.reportedResults) != 1 {
		t.Fatalf("terminal persistence calls = %d, want 1", len(tasks.reportedResults))
	}
	if len(stream.sent) != 1 || stream.sent[0].GetSessionReady() == nil {
		t.Fatalf("stale stream received terminal acknowledgement after takeover: %#v", stream.sent)
	}
}

func TestExpiredAttachedSessionRejectsOperationalHandlersBeforeSideEffects(t *testing.T) {
	type fixture struct {
		agent     *agentdomain.Agent
		session   ActiveControlSession
		service   *ControlPlaneService
		lifecycle *runtimeLifecycleStub
		claims    *executionClaimRuntimeStub
		tasks     *taskRuntimeStub
		stream    *fakeConnectStream
	}
	newFixture := func(t *testing.T) fixture {
		t.Helper()
		now := time.Date(2026, 7, 22, 13, 0, 0, 0, time.UTC)
		registry := newActiveSessionRegistry(time.Minute, func() time.Time { return now })
		agent := &agentdomain.Agent{ID: 78, InstanceID: "agt-78"}
		session, _ := registry.Register(agent.ID, 780, "session-78")
		session, ok := registry.MarkReadyIfCurrent(agent.ID, session.SessionID, session.SessionEpoch, session.StreamID)
		if !ok {
			t.Fatal("mark expired fixture ready")
		}
		lifecycle := &runtimeLifecycleStub{}
		claims := &executionClaimRuntimeStub{}
		tasks := &taskRuntimeStub{}
		service := NewControlPlaneService(&agentFinderStub{agent: agent}, lifecycle, tasks)
		service.executionClaims = claims
		service.WithActiveSessionRegistry(registry)
		now = now.Add(time.Minute + time.Nanosecond)
		return fixture{agent: agent, session: session, service: service, lifecycle: lifecycle, claims: claims, tasks: tasks, stream: &fakeConnectStream{ctx: context.Background()}}
	}

	t.Run("heartbeat", func(t *testing.T) {
		f := newFixture(t)
		err := f.service.handleHeartbeat(context.Background(), f.agent, f.session, &agentcontrolv1.Heartbeat{
			Agent: resourcenames.Agent(f.agent.InstanceID), Session: resourcenames.AgentSession(f.agent.InstanceID, f.session.SessionID), CompatibilityRevision: testCompatibilityRevision,
		})
		if status.Code(err) != codes.FailedPrecondition || len(f.lifecycle.heartbeats) != 0 {
			t.Fatalf("expired heartbeat = %v, persisted=%d", err, len(f.lifecycle.heartbeats))
		}
	})

	t.Run("request task", func(t *testing.T) {
		f := newFixture(t)
		err := f.service.handleExecutionPlanRequest(context.Background(), f.agent, f.session, &sync.Mutex{}, f.stream, &agentcontrolv1.RequestTask{
			RequestId: "550e8400-e29b-41d4-a716-446655440000",
		})
		if status.Code(err) != codes.FailedPrecondition || f.claims.calls != 0 || len(f.stream.sent) != 0 {
			t.Fatalf("expired request task = %v, claims=%d sent=%d", err, f.claims.calls, len(f.stream.sent))
		}
	})

	t.Run("terminal result", func(t *testing.T) {
		f := newFixture(t)
		err := f.service.handleTerminalTaskResult(context.Background(), f.agent, f.session, &sync.Mutex{}, f.stream, &agentcontrolv1.TerminalTaskResult{
			Task: resourcenames.Task(70, 701), Result: agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_SUCCEEDED, CompatibilityRevision: testCompatibilityRevision,
		})
		if status.Code(err) != codes.FailedPrecondition || len(f.tasks.reportedResults) != 0 || len(f.stream.sent) != 0 {
			t.Fatalf("expired terminal result = %v, reports=%d sent=%d", err, len(f.tasks.reportedResults), len(f.stream.sent))
		}
	})
}
