package agentcontrol

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestConnectRejectsMissingAgentAuthenticationToken(t *testing.T) {
	svc := NewControlPlaneService(
		&agentFinderStub{agent: &agentdomain.Agent{ID: 1}},
		&runtimeLifecycleStub{},
		&taskRuntimeStub{},
	)

	stream := &fakeConnectStream{ctx: context.Background()}
	err := svc.Connect(stream)
	if code := status.Code(err); code != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got=%s err=%v", code, err)
	}
}

func TestConnectRejectsForgedPublicSourceWithoutAgentAuthenticationToken(t *testing.T) {
	finder := &agentFinderStub{agent: &agentdomain.Agent{ID: 1}}
	lifecycle := &runtimeLifecycleStub{}
	svc := NewControlPlaneService(finder, lifecycle, &taskRuntimeStub{})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		forwardedForMetadataKey, "8.8.8.8",
	))
	err := svc.Connect(&fakeConnectStream{ctx: ctx})
	if code := status.Code(err); code != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got=%s err=%v", code, err)
	}
	if finder.lastAuthenticationToken != "" || lifecycle.connectedCount != 0 {
		t.Fatalf("forged source observation reached an admission path: token=%q connected=%d", finder.lastAuthenticationToken, lifecycle.connectedCount)
	}
}

func TestConnectRestoresFreshSameProcessSessionEpochFromPersistedRuntimeState(t *testing.T) {
	lastHeartbeat := time.Now().UTC().Add(-time.Second)
	agent := &agentdomain.Agent{
		ID:            8,
		InstanceID:    "agt-8",
		SessionID:     "session-8",
		SessionEpoch:  2,
		LastHeartbeat: &lastHeartbeat,
		MaxTasks:      3,
		CPUThreshold:  80,
		MemThreshold:  85,
		DiskThreshold: 90,
	}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	tasks := &taskRuntimeStub{}
	svc := NewControlPlaneService(finder, lifecycle, tasks)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-8"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*agentcontrolv1.ConnectRequest{
			runtimeRegisterSessionRequest("agt-8", "session-8"),
		},
	}

	if err := svc.Connect(stream); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	ready := stream.sent[0].GetSessionReady()
	if ready == nil || ready.GetSessionEpoch() != 2 {
		t.Fatalf("expected restored session epoch 2, got %+v", ready)
	}
	if len(tasks.fencedEpochs) != 1 || tasks.fencedEpochs[0] != 2 {
		t.Fatalf("expected synchronous fence at restored epoch 2, got %+v", tasks.fencedEpochs)
	}
}

func TestConnectAllocatesHigherEpochForStaleOrDifferentPersistedProcess(t *testing.T) {
	for _, test := range []struct {
		name          string
		persistedID   string
		lastHeartbeat time.Time
	}{
		{name: "stale same process", persistedID: "session-8", lastHeartbeat: time.Now().UTC().Add(-DefaultSessionHeartbeatTimeout - time.Second)},
		{name: "different fresh process", persistedID: "session-old", lastHeartbeat: time.Now().UTC().Add(-time.Second)},
	} {
		t.Run(test.name, func(t *testing.T) {
			agent := &agentdomain.Agent{ID: 8, InstanceID: "agt-8", SessionID: test.persistedID, SessionEpoch: 2, LastHeartbeat: &test.lastHeartbeat, MaxTasks: 3, CPUThreshold: 80, MemThreshold: 85, DiskThreshold: 90}
			tasks := &taskRuntimeStub{}
			service := NewControlPlaneService(&agentFinderStub{agent: agent}, &runtimeLifecycleStub{}, tasks)
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-8"))
			stream := &fakeConnectStream{ctx: ctx, incoming: []*agentcontrolv1.ConnectRequest{runtimeRegisterSessionRequest("agt-8", "session-8")}}
			if err := service.Connect(stream); err != nil {
				t.Fatalf("Connect() error = %v", err)
			}
			ready := stream.sent[0].GetSessionReady()
			if ready == nil || ready.GetSessionEpoch() != 3 {
				t.Fatalf("replacement epoch = %#v, want 3", ready)
			}
			if len(tasks.fencedEpochs) != 1 || tasks.fencedEpochs[0] != 3 {
				t.Fatalf("synchronous fence epochs = %v, want [3]", tasks.fencedEpochs)
			}
		})
	}
}

func TestConnectUsesObservedPublicSourceForLifecycle(t *testing.T) {
	agent := &agentdomain.Agent{ID: 19, InstanceID: "agt-19"}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	svc := NewControlPlaneService(finder, lifecycle, &taskRuntimeStub{})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-19",
		forwardedForMetadataKey, "8.8.8.8",
	))
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: &net.TCPAddr{
			IP:   net.ParseIP("8.8.8.8"),
			Port: 50001,
		},
	})
	stream := &fakeConnectStream{ctx: ctx}
	stream.incoming = []*agentcontrolv1.ConnectRequest{runtimeRegisterSessionRequest("agt-19", "session-19")}

	if err := svc.Connect(stream); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	if lifecycle.connectedIP != "8.8.8.8" {
		t.Fatalf("expected observed public source 8.8.8.8, got %q", lifecycle.connectedIP)
	}
}

func TestConnectHandlesHeartbeatAndTerminalTaskResult(t *testing.T) {
	agent := &agentdomain.Agent{ID: 9, InstanceID: "agt-9"}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	tasks := &taskRuntimeStub{}
	svc := NewControlPlaneService(finder, lifecycle, tasks)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-9"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*agentcontrolv1.ConnectRequest{
			runtimeRegisterSessionRequest("agt-9", "session-1"),
			{
				Payload: &agentcontrolv1.ConnectRequest_Heartbeat{
					Heartbeat: &agentcontrolv1.Heartbeat{
						Agent:                 resourcenames.Agent("agt-9"),
						Session:               resourcenames.AgentSession("agt-9", "session-1"),
						ObservedHostname:      "agent-1",
						CpuUsage:              21.5,
						MemUsage:              35.2,
						DiskUsage:             41.3,
						RunningTasks:          2,
						TaskSlotsUsed:         3,
						AgentVersion:          "v1.2.3",
						UptimeSeconds:         1234,
						CompatibilityRevision: testCompatibilityRevision,
						Health: &agentcontrolv1.HealthStatus{
							State: agentcontrolv1.HealthState_HEALTH_STATE_HEALTHY,
						},
					},
				},
			},
			{
				Payload: &agentcontrolv1.ConnectRequest_TerminalTaskResult{
					TerminalTaskResult: &agentcontrolv1.TerminalTaskResult{
						Task:                  resourcenames.Task(12, 77),
						Result:                agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_FAILED,
						CompatibilityRevision: testCompatibilityRevision,
						Failure: &agentcontrolv1.FailureDetail{
							Kind:           "runtime_error",
							Message:        "boom",
							DisplayMessage: "Restore Agent storage permissions.",
						},
						Diagnostics: terminalDiagnosticsProtoForTest(),
					},
				},
			},
		},
	}

	if err := svc.Connect(stream); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	if len(lifecycle.heartbeats) != 2 {
		t.Fatalf("expected registration and heartbeat snapshots, got=%d", len(lifecycle.heartbeats))
	}
	heartbeat := lifecycle.heartbeats[1]
	if heartbeat.Health == nil || heartbeat.Health.State != "healthy" {
		t.Fatalf("expected healthy heartbeat mapping, got %+v", heartbeat)
	}
	if heartbeat.RunningTasks != 2 {
		t.Fatalf("expected running tasks to map from heartbeat, got %d", heartbeat.RunningTasks)
	}
	if heartbeat.TaskSlotsUsed != 3 {
		t.Fatalf("expected task slots used to map from heartbeat, got %d", heartbeat.TaskSlotsUsed)
	}
	if len(tasks.reportedResults) != 1 {
		t.Fatalf("expected terminal task result report once, got=%d", len(tasks.reportedResults))
	}
	if tasks.reportedResults[0].taskID != 77 || tasks.reportedResults[0].status != "failed" {
		t.Fatalf("unexpected terminal task result report payload: %+v", tasks.reportedResults[0])
	}
	if tasks.reportedResults[0].sessionEpoch <= 0 {
		t.Fatalf("expected terminal task result to carry active session epoch, got %+v", tasks.reportedResults[0])
	}
	if tasks.reportedResults[0].sessionID != "session-1" {
		t.Fatalf("expected terminal task result to carry active process session, got %+v", tasks.reportedResults[0])
	}
	if tasks.reportedResults[0].failure == nil || tasks.reportedResults[0].failure.Kind != "runtime_error" || tasks.reportedResults[0].failure.Message != "boom" || tasks.reportedResults[0].failure.DisplayMessage != "Restore Agent storage permissions." {
		t.Fatalf("unexpected failure payload: %+v", tasks.reportedResults[0])
	}
}

func TestConnectRecordsHeartbeatThroughExplicitLifecycleAction(t *testing.T) {
	agent := &agentdomain.Agent{ID: 91, InstanceID: "agt-91"}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	svc := NewControlPlaneService(finder, lifecycle, &taskRuntimeStub{})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-91"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*agentcontrolv1.ConnectRequest{
			runtimeRegisterSessionRequest("agt-91", "session-91"),
			{
				Payload: &agentcontrolv1.ConnectRequest_Heartbeat{
					Heartbeat: &agentcontrolv1.Heartbeat{
						Agent:                 resourcenames.Agent("agt-91"),
						Session:               resourcenames.AgentSession("agt-91", "session-91"),
						ObservedHostname:      "node-91",
						AgentVersion:          "v9.1.0",
						CompatibilityRevision: testCompatibilityRevision,
						Health: &agentcontrolv1.HealthStatus{
							State: agentcontrolv1.HealthState_HEALTH_STATE_HEALTHY,
						},
					},
				},
			},
		},
	}

	if err := svc.Connect(stream); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	if len(lifecycle.heartbeats) != 2 {
		t.Fatalf("expected registration and explicit heartbeat lifecycle actions, got=%d", len(lifecycle.heartbeats))
	}
	if lifecycle.heartbeats[1].ObservedHostname != "node-91" {
		t.Fatalf("unexpected heartbeat payload: %+v", lifecycle.heartbeats[1])
	}
}

func TestConnectReportsTerminalTaskResultThroughExplicitBridgeAction(t *testing.T) {
	agent := &agentdomain.Agent{ID: 92, InstanceID: "agt-92"}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	tasks := &taskRuntimeStub{}
	svc := NewControlPlaneService(finder, lifecycle, tasks)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-92"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*agentcontrolv1.ConnectRequest{
			runtimeRegisterSessionRequest("agt-92", "session-92"),
			{
				Payload: &agentcontrolv1.ConnectRequest_TerminalTaskResult{
					TerminalTaskResult: &agentcontrolv1.TerminalTaskResult{
						Task:                  resourcenames.Task(12, 109),
						Result:                agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_FAILED,
						CompatibilityRevision: testCompatibilityRevision,
						Failure: &agentcontrolv1.FailureDetail{
							Kind:    "runtime_error",
							Message: "boom",
						},
						Diagnostics: terminalDiagnosticsProtoForTest(),
					},
				},
			},
		},
	}

	if err := svc.Connect(stream); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	if len(tasks.reportedResults) != 1 {
		t.Fatalf("expected explicit terminal task result bridge action once, got=%d", len(tasks.reportedResults))
	}
	if tasks.reportedResults[0].taskID != 109 || tasks.reportedResults[0].status != "failed" {
		t.Fatalf("unexpected terminal task result report payload: %+v", tasks.reportedResults[0])
	}
	if len(stream.sent) != 2 {
		t.Fatalf("expected SessionReady then terminal acknowledgement, got %d events", len(stream.sent))
	}
	ack := stream.sent[1].GetTerminalTaskResultAck()
	if ack == nil || ack.GetTask() != resourcenames.Task(12, 109) || ack.GetSessionEpoch() != tasks.reportedResults[0].sessionEpoch || ack.GetCompatibilityRevision() != testCompatibilityRevision {
		t.Fatalf("terminal acknowledgement = %+v, want task and persisted session epoch", ack)
	}
}

func TestConnectRejectsOperationalMessagesBeforeRegisterSession(t *testing.T) {
	agent := &agentdomain.Agent{ID: 18, InstanceID: "agt-18"}
	finder := &agentFinderStub{agent: agent}
	svc := NewControlPlaneService(finder, &runtimeLifecycleStub{}, &taskRuntimeStub{})

	cases := []struct {
		name string
		req  *agentcontrolv1.ConnectRequest
	}{
		{
			name: "heartbeat",
			req: &agentcontrolv1.ConnectRequest{
				Payload: &agentcontrolv1.ConnectRequest_Heartbeat{
					Heartbeat: &agentcontrolv1.Heartbeat{
						Agent:                 resourcenames.Agent("agt-18"),
						Session:               resourcenames.AgentSession("agt-18", "session-18"),
						ObservedHostname:      "node-18",
						CompatibilityRevision: testCompatibilityRevision,
					},
				},
			},
		},
		{
			name: "request task",
			req: &agentcontrolv1.ConnectRequest{
				Payload: &agentcontrolv1.ConnectRequest_RequestTask{RequestTask: &agentcontrolv1.RequestTask{
					RequestId: "550e8400-e29b-41d4-a716-446655440000",
				}},
			},
		},
		{
			name: "terminal task result",
			req: &agentcontrolv1.ConnectRequest{
				Payload: &agentcontrolv1.ConnectRequest_TerminalTaskResult{
					TerminalTaskResult: &agentcontrolv1.TerminalTaskResult{
						Task:                  resourcenames.Task(1, 1),
						Result:                agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_SUCCEEDED,
						CompatibilityRevision: testCompatibilityRevision,
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-18"))
			stream := &fakeConnectStream{
				ctx:      ctx,
				incoming: []*agentcontrolv1.ConnectRequest{tc.req},
			}

			err := svc.Connect(stream)
			if code := status.Code(err); code != codes.FailedPrecondition {
				t.Fatalf("expected failed precondition, got=%s err=%v", code, err)
			}
		})
	}
}

func TestConnectRejectsUnspecifiedHeartbeatHealthState(t *testing.T) {
	agent := &agentdomain.Agent{ID: 19, InstanceID: "agt-19"}
	finder := &agentFinderStub{agent: agent}
	svc := NewControlPlaneService(finder, &runtimeLifecycleStub{}, &taskRuntimeStub{})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-19"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*agentcontrolv1.ConnectRequest{
			runtimeRegisterSessionRequest("agt-19", "session-19"),
			{
				Payload: &agentcontrolv1.ConnectRequest_Heartbeat{
					Heartbeat: &agentcontrolv1.Heartbeat{
						Agent:                 resourcenames.Agent("agt-19"),
						Session:               resourcenames.AgentSession("agt-19", "session-19"),
						ObservedHostname:      "node-19",
						CompatibilityRevision: testCompatibilityRevision,
						Health: &agentcontrolv1.HealthStatus{
							State: agentcontrolv1.HealthState_HEALTH_STATE_UNSPECIFIED,
						},
					},
				},
			},
		},
	}

	err := svc.Connect(stream)
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got=%s err=%v", code, err)
	}
}

func TestConnectRejectsUnspecifiedTerminalTaskResultState(t *testing.T) {
	agent := &agentdomain.Agent{ID: 20, InstanceID: "agt-20"}
	finder := &agentFinderStub{agent: agent}
	svc := NewControlPlaneService(finder, &runtimeLifecycleStub{}, &taskRuntimeStub{})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-20"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*agentcontrolv1.ConnectRequest{
			runtimeRegisterSessionRequest("agt-20", "session-20"),
			{
				Payload: &agentcontrolv1.ConnectRequest_TerminalTaskResult{
					TerminalTaskResult: &agentcontrolv1.TerminalTaskResult{
						Task:                  resourcenames.Task(12, 88),
						Result:                agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_UNSPECIFIED,
						CompatibilityRevision: testCompatibilityRevision,
					},
				},
			},
		},
	}

	err := svc.Connect(stream)
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got=%s err=%v", code, err)
	}
}

func TestConnectRejectsMismatchedHeartbeatInstanceID(t *testing.T) {
	agent := &agentdomain.Agent{ID: 21, InstanceID: "agt-expected"}
	finder := &agentFinderStub{agent: agent}
	svc := NewControlPlaneService(finder, &runtimeLifecycleStub{}, &taskRuntimeStub{})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-21"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*agentcontrolv1.ConnectRequest{
			runtimeRegisterSessionRequest("agt-expected", "session-1"),
			{
				Payload: &agentcontrolv1.ConnectRequest_Heartbeat{
					Heartbeat: &agentcontrolv1.Heartbeat{
						Agent:                 resourcenames.Agent("agt-other"),
						Session:               resourcenames.AgentSession("agt-other", "session-1"),
						ObservedHostname:      "node-a",
						CompatibilityRevision: testCompatibilityRevision,
						Health: &agentcontrolv1.HealthStatus{
							State: agentcontrolv1.HealthState_HEALTH_STATE_HEALTHY,
						},
					},
				},
			},
		},
	}

	err := svc.Connect(stream)
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got=%s err=%v", code, err)
	}
}

func TestConnectReceivesPublishedDownlinkEvents(t *testing.T) {
	agent := &agentdomain.Agent{ID: 33, InstanceID: "agt-33"}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	tasks := &taskRuntimeStub{}
	registry := NewAgentStreamRegistry()
	svc := NewControlPlaneService(finder, lifecycle, tasks, registry)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-33"))
	stream := newBlockingConnectStream(ctx)
	done := make(chan error, 1)
	go func() {
		done <- svc.Connect(stream)
	}()
	stream.recvCh <- runtimeRegisterSessionRequest("agt-33", "session-33")

	first := stream.mustRecvEvent(t)
	if first.GetSessionReady() == nil {
		t.Fatalf("expected first event to be session_ready, got %+v", first)
	}

	publisher := NewAgentControlEventPublisher(registry)
	publisher.SendTaskCancel(33, 12, 88)
	cancelEvent := stream.mustRecvEvent(t)
	if cancelEvent.GetTaskCancel() == nil || cancelEvent.GetTaskCancel().GetTask() != resourcenames.Task(12, 88) {
		t.Fatalf("unexpected task cancel event: %+v", cancelEvent)
	}

	delivered := publisher.SendUpdateRequired(33, agentdomain.UpdateRequiredPayload{
		AgentVersion:  "v2.1.0",
		AgentImageRef: "registry.example.com/lunafox-agent:v2.1.0",
	})
	if !delivered {
		t.Fatalf("expected update_required event delivery")
	}
	updateEvent := stream.mustRecvEvent(t)
	if updateEvent.GetUpdateRequired() == nil || updateEvent.GetUpdateRequired().GetAgentVersion() != "v2.1.0" {
		t.Fatalf("unexpected update required event: %+v", updateEvent)
	}
	if updateEvent.GetUpdateRequired().GetAgentImage() != "registry.example.com/lunafox-agent:v2.1.0" {
		t.Fatalf("unexpected update required agent image ref: %+v", updateEvent)
	}

	stream.pushRecvError(io.EOF)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected clean stream close, got=%v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting connect to exit")
	}
}

func TestConnectDropsPublishedDownlinkUntilReplacementSessionReady(t *testing.T) {
	agent := &agentdomain.Agent{ID: 34, InstanceID: "agt-34"}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &blockingOnConnectedLifecycleStub{
		blockOnCall: 2,
		blockedCh:   make(chan struct{}),
		releaseCh:   make(chan struct{}),
	}
	tasks := &taskRuntimeStub{}
	registry := NewAgentStreamRegistry()
	svc := NewControlPlaneService(finder, lifecycle, tasks, registry)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-34"))

	first := newBlockingConnectStream(ctx)
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- svc.Connect(first)
	}()
	first.recvCh <- runtimeRegisterSessionRequest("agt-34", "session-old")
	if event := first.mustRecvEvent(t); event.GetSessionReady() == nil {
		t.Fatalf("expected first session_ready event, got %+v", event)
	}

	second := newBlockingConnectStream(ctx)
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- svc.Connect(second)
	}()
	second.recvCh <- runtimeRegisterSessionRequest("agt-34", "session-new")

	select {
	case <-lifecycle.blockedCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for replacement OnConnected to block")
	}

	pendingCancel := &agentcontrolv1.ConnectResponse{
		Payload: &agentcontrolv1.ConnectResponse_TaskCancel{
			TaskCancel: &agentcontrolv1.TaskCancel{Task: resourcenames.Task(12, 99)},
		},
	}
	if delivered := registry.Publish(34, pendingCancel); delivered {
		t.Fatal("expected pending replacement stream not to receive task_cancel before session_ready")
	}

	select {
	case event := <-second.sentCh:
		t.Fatalf("expected replacement stream to wait for session_ready, got %+v", event)
	case <-time.After(150 * time.Millisecond):
	}

	select {
	case event := <-first.sentCh:
		t.Fatalf("expected replaced stream not to receive task_cancel during replacement handshake, got %+v", event)
	case <-time.After(150 * time.Millisecond):
	}

	close(lifecycle.releaseCh)

	if event := second.mustRecvEvent(t); event.GetSessionReady() == nil {
		t.Fatalf("expected replacement session_ready event after release, got %+v", event)
	}

	readyCancel := &agentcontrolv1.ConnectResponse{
		Payload: &agentcontrolv1.ConnectResponse_TaskCancel{
			TaskCancel: &agentcontrolv1.TaskCancel{Task: resourcenames.Task(12, 100)},
		},
	}
	if delivered := registry.Publish(34, readyCancel); !delivered {
		t.Fatal("expected task_cancel delivery after replacement session becomes ready")
	}
	if event := second.mustRecvEvent(t); event.GetTaskCancel() == nil || event.GetTaskCancel().GetTask() != resourcenames.Task(12, 100) {
		t.Fatalf("unexpected task_cancel after replacement ready: %+v", event)
	}

	first.pushRecvError(io.EOF)
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("expected stale stream close without error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for stale stream close")
	}

	second.pushRecvError(io.EOF)
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("expected replacement stream close without error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for replacement stream close")
	}
}

func TestConnectDropsPublishedDownlinkUntilReconnectSessionReady(t *testing.T) {
	agent := &agentdomain.Agent{ID: 35, InstanceID: "agt-35"}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &blockingOnConnectedLifecycleStub{
		blockOnCall: 2,
		blockedCh:   make(chan struct{}),
		releaseCh:   make(chan struct{}),
	}
	tasks := &taskRuntimeStub{}
	registry := NewAgentStreamRegistry()
	svc := NewControlPlaneService(finder, lifecycle, tasks, registry)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-35"))

	first := newBlockingConnectStream(ctx)
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- svc.Connect(first)
	}()
	first.recvCh <- runtimeRegisterSessionRequest("agt-35", "session-35")
	firstReady := first.mustRecvEvent(t)
	if firstReady.GetSessionReady() == nil {
		t.Fatalf("expected first session_ready event, got %+v", firstReady)
	}
	firstEpoch := firstReady.GetSessionReady().GetSessionEpoch()
	if firstEpoch <= 0 {
		t.Fatalf("expected positive first session epoch, got %d", firstEpoch)
	}

	second := newBlockingConnectStream(ctx)
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- svc.Connect(second)
	}()
	second.recvCh <- runtimeRegisterSessionRequest("agt-35", "session-35")

	select {
	case <-lifecycle.blockedCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for reconnect OnConnected to block")
	}

	pendingCancel := &agentcontrolv1.ConnectResponse{
		Payload: &agentcontrolv1.ConnectResponse_TaskCancel{
			TaskCancel: &agentcontrolv1.TaskCancel{Task: resourcenames.Task(12, 199)},
		},
	}
	if delivered := registry.Publish(35, pendingCancel); delivered {
		t.Fatal("expected reconnecting stream not to receive task_cancel before session_ready")
	}

	select {
	case event := <-second.sentCh:
		t.Fatalf("expected reconnecting stream to wait for session_ready, got %+v", event)
	case <-time.After(150 * time.Millisecond):
	}

	select {
	case event := <-first.sentCh:
		t.Fatalf("expected previous transport stream not to receive task_cancel during reconnect handshake, got %+v", event)
	case <-time.After(150 * time.Millisecond):
	}

	close(lifecycle.releaseCh)

	secondReady := second.mustRecvEvent(t)
	if secondReady.GetSessionReady() == nil {
		t.Fatalf("expected reconnect session_ready event after release, got %+v", secondReady)
	}
	if secondReady.GetSessionReady().GetSessionEpoch() != firstEpoch {
		t.Fatalf("expected reconnect to reuse session epoch %d, got %d", firstEpoch, secondReady.GetSessionReady().GetSessionEpoch())
	}

	readyCancel := &agentcontrolv1.ConnectResponse{
		Payload: &agentcontrolv1.ConnectResponse_TaskCancel{
			TaskCancel: &agentcontrolv1.TaskCancel{Task: resourcenames.Task(12, 200)},
		},
	}
	if delivered := registry.Publish(35, readyCancel); !delivered {
		t.Fatal("expected task_cancel delivery after reconnecting stream becomes ready")
	}
	if event := second.mustRecvEvent(t); event.GetTaskCancel() == nil || event.GetTaskCancel().GetTask() != resourcenames.Task(12, 200) {
		t.Fatalf("unexpected task_cancel after reconnect ready: %+v", event)
	}

	first.pushRecvError(io.EOF)
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("expected previous transport stream close without error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for previous transport stream close")
	}

	second.pushRecvError(io.EOF)
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("expected reconnecting stream close without error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for reconnecting stream close")
	}
}

func TestConnectRejectsHeartbeatFromStaleSession(t *testing.T) {
	agent := &agentdomain.Agent{ID: 54, InstanceID: "agt-54"}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	svc := NewControlPlaneService(finder, lifecycle, &taskRuntimeStub{}, NewAgentStreamRegistry())

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-54"))
	first := newBlockingConnectStream(ctx)
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- svc.Connect(first)
	}()
	first.recvCh <- runtimeRegisterSessionRequest("agt-54", "session-old")
	if event := first.mustRecvEvent(t); event.GetSessionReady() == nil {
		t.Fatalf("expected first session_ready event, got %+v", event)
	}

	second := newBlockingConnectStream(ctx)
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- svc.Connect(second)
	}()
	second.recvCh <- runtimeRegisterSessionRequest("agt-54", "session-new")
	if event := second.mustRecvEvent(t); event.GetSessionReady() == nil {
		t.Fatalf("expected second session_ready event, got %+v", event)
	}

	first.recvCh <- &agentcontrolv1.ConnectRequest{
		Payload: &agentcontrolv1.ConnectRequest_Heartbeat{
			Heartbeat: &agentcontrolv1.Heartbeat{
				Agent:                 resourcenames.Agent("agt-54"),
				Session:               resourcenames.AgentSession("agt-54", "session-old"),
				ObservedHostname:      "node-old",
				CompatibilityRevision: testCompatibilityRevision,
				Health: &agentcontrolv1.HealthStatus{
					State: agentcontrolv1.HealthState_HEALTH_STATE_HEALTHY,
				},
			},
		},
	}

	select {
	case err := <-firstDone:
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("expected stale session rejection, got err=%v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for stale session rejection")
	}

	second.pushRecvError(io.EOF)
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("expected active session close without error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting active session close")
	}
}

func TestConnectDefersOfflineTransitionForTransportDisconnects(t *testing.T) {
	agent := &agentdomain.Agent{ID: 55, InstanceID: "agt-55"}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	svc := NewControlPlaneService(finder, lifecycle, &taskRuntimeStub{}, NewAgentStreamRegistry())

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-55"))
	first := newBlockingConnectStream(ctx)
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- svc.Connect(first)
	}()
	first.recvCh <- runtimeRegisterSessionRequest("agt-55", "session-old")
	if event := first.mustRecvEvent(t); event.GetSessionReady() == nil {
		t.Fatalf("expected first session_ready event, got %+v", event)
	}

	second := newBlockingConnectStream(ctx)
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- svc.Connect(second)
	}()
	second.recvCh <- runtimeRegisterSessionRequest("agt-55", "session-new")
	if event := second.mustRecvEvent(t); event.GetSessionReady() == nil {
		t.Fatalf("expected second session_ready event, got %+v", event)
	}

	first.pushRecvError(io.EOF)
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("expected stale session EOF to close cleanly, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting stale session close")
	}
	if lifecycle.disconnectedCount != 0 || lifecycle.detachedCount != 0 {
		t.Fatalf("expected stale disconnect ignored, got offline=%d detached=%d", lifecycle.disconnectedCount, lifecycle.detachedCount)
	}

	second.pushRecvError(io.EOF)
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("expected active session EOF to close cleanly, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting active session close")
	}
	if lifecycle.disconnectedCount != 0 || lifecycle.disconnectedAgent != 0 {
		t.Fatalf("expected transport disconnects to defer offline transition, got count=%d agent=%d", lifecycle.disconnectedCount, lifecycle.disconnectedAgent)
	}
	if lifecycle.detachedCount != 1 || lifecycle.detachedAgent != 55 {
		t.Fatalf("expected active transport disconnect to clear its connection address once, got count=%d agent=%d", lifecycle.detachedCount, lifecycle.detachedAgent)
	}
}

func TestConnectSerializesAddressClearBeforeReplacementRegistration(t *testing.T) {
	agent := &agentdomain.Agent{ID: 56, InstanceID: "agt-56"}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &blockingOnDetachedLifecycleStub{
		detachStarted: make(chan struct{}),
		releaseDetach: make(chan struct{}),
	}
	svc := NewControlPlaneService(finder, lifecycle, &taskRuntimeStub{}, NewAgentStreamRegistry())
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-56"))

	first := newBlockingConnectStream(ctx)
	firstDone := make(chan error, 1)
	go func() { firstDone <- svc.Connect(first) }()
	first.recvCh <- runtimeRegisterSessionRequest("agt-56", "session-56")
	if event := first.mustRecvEvent(t); event.GetSessionReady() == nil {
		t.Fatalf("expected first session_ready event, got %+v", event)
	}

	first.pushRecvError(io.EOF)
	select {
	case <-lifecycle.detachStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for connection-address clear to begin")
	}

	second := newBlockingConnectStream(ctx)
	secondDone := make(chan error, 1)
	go func() { secondDone <- svc.Connect(second) }()
	second.recvCh <- runtimeRegisterSessionRequest("agt-56", "session-56")
	select {
	case event := <-second.sentCh:
		t.Fatalf("replacement registered before previous address clear finished: %+v", event)
	case <-time.After(100 * time.Millisecond):
	}

	close(lifecycle.releaseDetach)
	if event := second.mustRecvEvent(t); event.GetSessionReady() == nil {
		t.Fatalf("expected replacement session_ready event, got %+v", event)
	}
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first connection closed with %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first connection to close")
	}

	second.pushRecvError(io.EOF)
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("replacement connection closed with %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for replacement connection to close")
	}
}

func TestConnectSupportsReconnect(t *testing.T) {
	agent := &agentdomain.Agent{ID: 44, InstanceID: "agt-44"}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	tasks := &taskRuntimeStub{}
	svc := NewControlPlaneService(finder, lifecycle, tasks)

	var epochs []int64
	for i := 0; i < 2; i++ {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-44"))
		stream := &fakeConnectStream{ctx: ctx, incoming: []*agentcontrolv1.ConnectRequest{runtimeRegisterSessionRequest("agt-44", "session-44")}}
		if err := svc.Connect(stream); err != nil {
			t.Fatalf("connect #%d failed: %v", i+1, err)
		}
		if len(stream.sent) != 1 || stream.sent[0].GetSessionReady() == nil {
			t.Fatalf("expected session_ready event on connect #%d, got %+v", i+1, stream.sent)
		}
		epochs = append(epochs, stream.sent[0].GetSessionReady().GetSessionEpoch())
	}

	if lifecycle.connectedCount != 2 {
		t.Fatalf("expected 2 connects, got=%d", lifecycle.connectedCount)
	}
	if lifecycle.disconnectedCount != 0 {
		t.Fatalf("expected reconnects not to mark agent offline on stream close, got=%d", lifecycle.disconnectedCount)
	}
	if len(epochs) != 2 || epochs[0] != epochs[1] {
		t.Fatalf("expected same-session reconnect to reuse epoch, got=%v", epochs)
	}
}

func TestConnectAssignsNewEpochForDifferentSessionTakeover(t *testing.T) {
	agent := &agentdomain.Agent{ID: 46, InstanceID: "agt-46"}
	finder := &agentFinderStub{agent: agent}
	svc := NewControlPlaneService(finder, &runtimeLifecycleStub{}, &taskRuntimeStub{})

	var epochs []int64
	for _, sessionID := range []string{"session-old", "session-new"} {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-46"))
		stream := &fakeConnectStream{ctx: ctx, incoming: []*agentcontrolv1.ConnectRequest{runtimeRegisterSessionRequest("agt-46", sessionID)}}
		if err := svc.Connect(stream); err != nil {
			t.Fatalf("connect for %s failed: %v", sessionID, err)
		}
		if len(stream.sent) != 1 || stream.sent[0].GetSessionReady() == nil {
			t.Fatalf("expected session_ready for %s, got %+v", sessionID, stream.sent)
		}
		epochs = append(epochs, stream.sent[0].GetSessionReady().GetSessionEpoch())
	}

	if len(epochs) != 2 || epochs[1] <= epochs[0] {
		t.Fatalf("expected new session takeover to allocate newer epoch, got=%v", epochs)
	}
}

func TestConnectMapsInternalErrors(t *testing.T) {
	t.Run("on connected failure", func(t *testing.T) {
		svc := NewControlPlaneService(
			&agentFinderStub{agent: &agentdomain.Agent{ID: 1, InstanceID: "agt-1"}},
			&runtimeLifecycleStub{onConnectedErr: errors.New("connect failed")},
			&taskRuntimeStub{},
		)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-1"))
		err := svc.Connect(&fakeConnectStream{ctx: ctx, incoming: []*agentcontrolv1.ConnectRequest{runtimeRegisterSessionRequest("agt-1", "session-1")}})
		if status.Code(err) != codes.Internal {
			t.Fatalf("expected internal, got=%v", err)
		}
	})

	t.Run("heartbeat delegation failure", func(t *testing.T) {
		svc := NewControlPlaneService(
			&agentFinderStub{agent: &agentdomain.Agent{ID: 2, InstanceID: "agt-2"}},
			&runtimeLifecycleStub{handleErr: errors.New("heartbeat failed")},
			&taskRuntimeStub{},
		)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-2"))
		stream := &fakeConnectStream{
			ctx: ctx,
			incoming: []*agentcontrolv1.ConnectRequest{
				runtimeRegisterSessionRequest("agt-2", "session-2"),
				{Payload: &agentcontrolv1.ConnectRequest_Heartbeat{Heartbeat: &agentcontrolv1.Heartbeat{
					Agent: resourcenames.Agent("agt-2"), Session: resourcenames.AgentSession("agt-2", "session-2"), ObservedHostname: "node-a", CpuUsage: 10, CompatibilityRevision: testCompatibilityRevision,
				}}},
			},
		}
		err := svc.Connect(stream)
		if status.Code(err) != codes.Internal {
			t.Fatalf("expected internal, got=%v", err)
		}
	})

	t.Run("v2 claim failure", func(t *testing.T) {
		claims := &executionClaimRuntimeStub{claimErr: errors.New("claim failed")}
		svc := NewControlPlaneService(
			&agentFinderStub{agent: &agentdomain.Agent{ID: 3, InstanceID: "agt-3"}},
			&runtimeLifecycleStub{},
			claims,
		)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-3"))
		stream := &fakeConnectStream{
			ctx: ctx,
			incoming: []*agentcontrolv1.ConnectRequest{
				runtimeRegisterSessionRequest("agt-3", "session-3"),
				{Payload: &agentcontrolv1.ConnectRequest_RequestTask{RequestTask: &agentcontrolv1.RequestTask{
					RequestId: "550e8400-e29b-41d4-a716-446655440000",
				}}},
			},
		}
		err := svc.Connect(stream)
		if status.Code(err) != codes.Internal {
			t.Fatalf("expected internal, got=%v", err)
		}
		if claims.calls != 1 {
			t.Fatalf("expected one v2 claim attempt, got=%d", claims.calls)
		}
		if len(stream.sent) != 1 || stream.sent[0].GetSessionReady() == nil {
			t.Fatalf("claim failure must not emit TaskAssign: %+v", stream.sent)
		}
	})

	t.Run("task status update failure", func(t *testing.T) {
		svc := NewControlPlaneService(
			&agentFinderStub{agent: &agentdomain.Agent{ID: 4, InstanceID: "agt-4"}},
			&runtimeLifecycleStub{},
			&taskRuntimeStub{updateErr: errors.New("update failed")},
		)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-4"))
		stream := &fakeConnectStream{
			ctx: ctx,
			incoming: []*agentcontrolv1.ConnectRequest{
				runtimeRegisterSessionRequest("agt-4", "session-4"),
				{Payload: &agentcontrolv1.ConnectRequest_TerminalTaskResult{TerminalTaskResult: &agentcontrolv1.TerminalTaskResult{
					Task: resourcenames.Task(1, 1), Result: agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_FAILED, CompatibilityRevision: testCompatibilityRevision,
					Diagnostics: terminalDiagnosticsProtoForTest(),
				}}},
			},
		}
		err := svc.Connect(stream)
		if status.Code(err) != codes.Internal {
			t.Fatalf("expected internal, got=%v", err)
		}
		if len(stream.sent) != 1 || stream.sent[0].GetSessionReady() == nil {
			t.Fatalf("failed terminal persistence must not emit acknowledgement: %+v", stream.sent)
		}
	})
}

type fakeConnectStream struct {
	ctx      context.Context
	incoming []*agentcontrolv1.ConnectRequest
	sent     []*agentcontrolv1.ConnectResponse
}

func (s *fakeConnectStream) Context() context.Context { return s.ctx }

func (s *fakeConnectStream) Send(msg *agentcontrolv1.ConnectResponse) error {
	s.sent = append(s.sent, msg)
	return nil
}

func (s *fakeConnectStream) Recv() (*agentcontrolv1.ConnectRequest, error) {
	if len(s.incoming) == 0 {
		return nil, io.EOF
	}
	next := s.incoming[0]
	s.incoming = s.incoming[1:]
	return next, nil
}

func (s *fakeConnectStream) SetHeader(metadata.MD) error  { return nil }
func (s *fakeConnectStream) SendHeader(metadata.MD) error { return nil }
func (s *fakeConnectStream) SetTrailer(metadata.MD)       {}
func (s *fakeConnectStream) SendMsg(any) error            { return nil }
func (s *fakeConnectStream) RecvMsg(any) error            { return nil }

func runtimeRegisterSessionRequest(instanceID, sessionID string) *agentcontrolv1.ConnectRequest {
	return &agentcontrolv1.ConnectRequest{
		Payload: &agentcontrolv1.ConnectRequest_RegisterSession{
			RegisterSession: &agentcontrolv1.RegisterSession{
				Agent:                    resourcenames.Agent(instanceID),
				Session:                  resourcenames.AgentSession(instanceID, sessionID),
				ObservedHostname:         "node-" + sessionID,
				AgentVersion:             "v1.0.0",
				OperatingSystem:          "linux",
				Architecture:             "amd64",
				ContainerRuntimeReady:    true,
				SupportedEngineApiMajors: []uint32{2},
				CompatibilityRevision:    engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
			},
		},
	}
}

type blockingConnectStream struct {
	ctx     context.Context
	recvCh  chan *agentcontrolv1.ConnectRequest
	recvErr chan error
	sentCh  chan *agentcontrolv1.ConnectResponse
	mu      sync.Mutex
	sent    []*agentcontrolv1.ConnectResponse
}

func newBlockingConnectStream(ctx context.Context) *blockingConnectStream {
	return &blockingConnectStream{
		ctx:     ctx,
		recvCh:  make(chan *agentcontrolv1.ConnectRequest),
		recvErr: make(chan error, 1),
		sentCh:  make(chan *agentcontrolv1.ConnectResponse, 16),
	}
}

func (s *blockingConnectStream) Context() context.Context { return s.ctx }

func (s *blockingConnectStream) Send(event *agentcontrolv1.ConnectResponse) error {
	s.mu.Lock()
	s.sent = append(s.sent, event)
	s.mu.Unlock()
	s.sentCh <- event
	return nil
}

func (s *blockingConnectStream) Recv() (*agentcontrolv1.ConnectRequest, error) {
	select {
	case req := <-s.recvCh:
		return req, nil
	case err := <-s.recvErr:
		return nil, err
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

func (s *blockingConnectStream) pushRecvError(err error) {
	s.recvErr <- err
}

func (s *blockingConnectStream) mustRecvEvent(t *testing.T) *agentcontrolv1.ConnectResponse {
	t.Helper()
	select {
	case event := <-s.sentCh:
		return event
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for outbound control-plane event")
		return nil
	}
}

func (s *blockingConnectStream) SetHeader(metadata.MD) error  { return nil }
func (s *blockingConnectStream) SendHeader(metadata.MD) error { return nil }
func (s *blockingConnectStream) SetTrailer(metadata.MD)       {}
func (s *blockingConnectStream) SendMsg(any) error            { return nil }
func (s *blockingConnectStream) RecvMsg(any) error            { return nil }

type agentFinderStub struct {
	agent                   *agentdomain.Agent
	err                     error
	lastAuthenticationToken string
}

func (s *agentFinderStub) FindByAuthenticationToken(_ context.Context, authenticationToken string) (*agentdomain.Agent, error) {
	s.lastAuthenticationToken = authenticationToken
	return s.agent, s.err
}

type runtimeLifecycleStub struct {
	connected         bool
	disconnected      bool
	detached          bool
	heartbeats        []agentdomain.AgentHeartbeatEvent
	connectedCount    int
	disconnectedCount int
	detachedCount     int
	connectedIP       string
	disconnectedAgent int
	detachedAgent     int
	onConnectedErr    error
	onDisconnectedErr error
	handleErr         error
}

func (s *runtimeLifecycleStub) OnConnected(_ context.Context, _ *agentdomain.Agent, connectionIP string) error {
	if s.onConnectedErr != nil {
		return s.onConnectedErr
	}
	s.connected = true
	s.connectedCount++
	s.connectedIP = connectionIP
	return nil
}

func (s *runtimeLifecycleStub) OnDisconnected(_ context.Context, agentID int) error {
	if s.onDisconnectedErr != nil {
		return s.onDisconnectedErr
	}
	s.disconnected = true
	s.disconnectedCount++
	s.disconnectedAgent = agentID
	return nil
}

func (s *runtimeLifecycleStub) OnControlConnectionDetached(_ context.Context, agentID int) error {
	s.detached = true
	s.detachedCount++
	s.detachedAgent = agentID
	return nil
}

func (s *runtimeLifecycleStub) RecordHeartbeat(_ context.Context, _ int, event agentdomain.AgentHeartbeatEvent) error {
	if s.handleErr != nil {
		return s.handleErr
	}
	s.heartbeats = append(s.heartbeats, event)
	return nil
}

type blockingOnConnectedLifecycleStub struct {
	runtimeLifecycleStub
	blockOnCall int
	blockedCh   chan struct{}
	releaseCh   chan struct{}
}

type blockingOnDetachedLifecycleStub struct {
	runtimeLifecycleStub
	detachStarted chan struct{}
	releaseDetach chan struct{}
}

func (s *blockingOnDetachedLifecycleStub) OnControlConnectionDetached(ctx context.Context, agentID int) error {
	s.detached = true
	s.detachedCount++
	s.detachedAgent = agentID
	if s.detachedCount != 1 {
		return nil
	}
	close(s.detachStarted)
	select {
	case <-s.releaseDetach:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *blockingOnConnectedLifecycleStub) OnConnected(ctx context.Context, agent *agentdomain.Agent, connectionIP string) error {
	s.connected = true
	s.connectedCount++
	s.connectedIP = connectionIP
	if s.onConnectedErr != nil {
		return s.onConnectedErr
	}
	if s.blockOnCall > 0 && s.connectedCount == s.blockOnCall {
		if s.blockedCh != nil {
			close(s.blockedCh)
		}
		if s.releaseCh != nil {
			select {
			case <-s.releaseCh:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

type taskUpdate struct {
	agentID      int
	sessionID    string
	sessionEpoch int64
	taskID       int
	status       string
	failure      *scanapp.FailureDetail
}

type taskRuntimeStub struct {
	claimPlan         *agentexecutionv1.ResolvedEngineExecutionPlan
	claimErr          error
	claimCalls        int
	claimAgentID      int
	claimSessionID    string
	claimSessionEpoch int64
	claimRequestID    string
	claimSnapshot     agentdomain.AgentExecutionCapabilitySnapshot
	updateErr         error
	reportedResults   []taskUpdate
	reportHook        func()
	fenceErr          error
	fencedEpochs      []int64
}

func (s *taskRuntimeStub) ClaimNextExecutionPlan(_ context.Context, agentID int, sessionID string, sessionEpoch int64, requestID string, snapshot agentdomain.AgentExecutionCapabilitySnapshot) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	s.claimCalls++
	s.claimAgentID = agentID
	s.claimSessionID = sessionID
	s.claimSessionEpoch = sessionEpoch
	s.claimRequestID = requestID
	s.claimSnapshot = snapshot.Clone()
	return s.claimPlan, s.claimErr
}

func (s *taskRuntimeStub) ReportTerminalTaskResult(_ context.Context, agentID int, sessionID string, sessionEpoch int64, taskID int, status string, failure *scanapp.FailureDetail) error {
	s.reportedResults = append(s.reportedResults, taskUpdate{agentID: agentID, sessionID: sessionID, sessionEpoch: sessionEpoch, taskID: taskID, status: status, failure: failure})
	if s.reportHook != nil {
		s.reportHook()
	}
	return s.updateErr
}

func (s *taskRuntimeStub) ReportTerminalTaskResultWithDiagnostics(_ context.Context, agentID int, sessionID string, sessionEpoch int64, taskID int, status string, failure *scanapp.FailureDetail, _ *scanapp.EngineExecutionDiagnostics) error {
	s.reportedResults = append(s.reportedResults, taskUpdate{agentID: agentID, sessionID: sessionID, sessionEpoch: sessionEpoch, taskID: taskID, status: status, failure: failure})
	if s.reportHook != nil {
		s.reportHook()
	}
	return s.updateErr
}

func terminalDiagnosticsProtoForTest() *engineexecutionpb.EngineExecutionDiagnostics {
	return &engineexecutionpb.EngineExecutionDiagnostics{
		CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
		Availability:          engineexecutionpb.EngineExecutionDiagnosticAvailability_ENGINE_EXECUTION_DIAGNOSTIC_AVAILABILITY_UNAVAILABLE,
		ResultState:           engineexecutionpb.EngineExecutionResultState_ENGINE_EXECUTION_RESULT_STATE_UNKNOWN,
	}
}

func (s *taskRuntimeStub) FenceSupersededAgentSession(_ context.Context, _ int, currentSessionEpoch int64) error {
	s.fencedEpochs = append(s.fencedEpochs, currentSessionEpoch)
	return s.fenceErr
}
