package agentcontrol

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type recordingConnectStream struct {
	ctx      context.Context
	sendErr  error
	sentCh   chan *agentcontrolv1.ConnectResponse
	mu       sync.Mutex
	sent     []*agentcontrolv1.ConnectResponse
	sendCall int
}

func newRecordingConnectStream(ctx context.Context) *recordingConnectStream {
	return &recordingConnectStream{
		ctx:    ctx,
		sentCh: make(chan *agentcontrolv1.ConnectResponse, 4),
	}
}

func (s *recordingConnectStream) Context() context.Context { return s.ctx }

func (s *recordingConnectStream) Send(event *agentcontrolv1.ConnectResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sendCall++
	if s.sendErr != nil {
		return s.sendErr
	}
	s.sent = append(s.sent, event)
	s.sentCh <- event
	return nil
}

func (s *recordingConnectStream) Recv() (*agentcontrolv1.ConnectRequest, error) { return nil, nil }
func (s *recordingConnectStream) SetHeader(metadata.MD) error                   { return nil }
func (s *recordingConnectStream) SendHeader(metadata.MD) error                  { return nil }
func (s *recordingConnectStream) SetTrailer(metadata.MD)                        {}
func (s *recordingConnectStream) SendMsg(any) error                             { return nil }
func (s *recordingConnectStream) RecvMsg(any) error                             { return nil }

type scriptedConnectStream struct {
	ctx          context.Context
	incoming     []*agentcontrolv1.ConnectRequest
	recvErr      error
	sendErrAfter int
	sendCount    int
	sent         []*agentcontrolv1.ConnectResponse
}

func (s *scriptedConnectStream) Context() context.Context { return s.ctx }

func (s *scriptedConnectStream) Send(event *agentcontrolv1.ConnectResponse) error {
	s.sendCount++
	if s.sendErrAfter > 0 && s.sendCount >= s.sendErrAfter {
		return errors.New("send failed")
	}
	s.sent = append(s.sent, event)
	return nil
}

func (s *scriptedConnectStream) Recv() (*agentcontrolv1.ConnectRequest, error) {
	if len(s.incoming) > 0 {
		next := s.incoming[0]
		s.incoming = s.incoming[1:]
		return next, nil
	}
	if s.recvErr != nil {
		return nil, s.recvErr
	}
	return nil, nil
}

func (s *scriptedConnectStream) SetHeader(metadata.MD) error  { return nil }
func (s *scriptedConnectStream) SendHeader(metadata.MD) error { return nil }
func (s *scriptedConnectStream) SetTrailer(metadata.MD)       {}
func (s *scriptedConnectStream) SendMsg(any) error            { return nil }
func (s *scriptedConnectStream) RecvMsg(any) error            { return nil }

func TestConnectReturnsUnimplementedWhenDependenciesMissing(t *testing.T) {
	svc := &ControlPlaneService{}

	err := svc.Connect(&fakeConnectStream{ctx: context.Background()})
	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("expected unimplemented, got %v", err)
	}
}

func TestConnectRejectsInvalidAuthenticatedAgent(t *testing.T) {
	testCases := []struct {
		name   string
		finder *agentFinderStub
	}{
		{name: "finder error", finder: &agentFinderStub{err: errors.New("lookup failed")}},
		{name: "nil agent", finder: &agentFinderStub{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewControlPlaneService(tc.finder, &runtimeLifecycleStub{}, &taskRuntimeStub{})
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-1"))

			err := svc.Connect(&fakeConnectStream{ctx: ctx})
			if status.Code(err) != codes.Unauthenticated {
				t.Fatalf("expected unauthenticated, got %v", err)
			}
		})
	}
}

func TestConnectReturnsRecvError(t *testing.T) {
	agent := &agentdomain.Agent{ID: 60, InstanceID: "agt-60"}
	svc := NewControlPlaneService(&agentFinderStub{agent: agent}, &runtimeLifecycleStub{}, &taskRuntimeStub{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-60"))
	stream := &scriptedConnectStream{ctx: ctx, recvErr: errors.New("recv failed")}

	err := svc.Connect(stream)
	if err == nil || err.Error() != "recv failed" {
		t.Fatalf("expected recv error to be returned, got %v", err)
	}
}

func TestConnectIgnoresNilAndUnknownPayloadRequests(t *testing.T) {
	agent := &agentdomain.Agent{ID: 61, InstanceID: "agt-61"}
	lifecycle := &runtimeLifecycleStub{}
	svc := NewControlPlaneService(&agentFinderStub{agent: agent}, lifecycle, &taskRuntimeStub{})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-61"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*agentcontrolv1.ConnectRequest{
			nil,
			runtimeRegisterSessionRequest("agt-61", "session-61"),
			{},
		},
	}

	if err := svc.Connect(stream); err != nil {
		t.Fatalf("expected nil/unknown requests to be ignored, got %v", err)
	}
	if lifecycle.connectedCount != 1 {
		t.Fatalf("expected one successful registration, got %d", lifecycle.connectedCount)
	}
	if len(stream.sent) != 1 || stream.sent[0].GetSessionReady() == nil {
		t.Fatalf("expected only session_ready response, got %+v", stream.sent)
	}
}

func TestConnectRejectsDuplicateRegisterSession(t *testing.T) {
	agent := &agentdomain.Agent{ID: 62, InstanceID: "agt-62"}
	svc := NewControlPlaneService(&agentFinderStub{agent: agent}, &runtimeLifecycleStub{}, &taskRuntimeStub{})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-62"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*agentcontrolv1.ConnectRequest{
			runtimeRegisterSessionRequest("agt-62", "session-62"),
			runtimeRegisterSessionRequest("agt-62", "session-62"),
		},
	}

	err := svc.Connect(stream)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected failed precondition, got %v", err)
	}
}

func TestConnectIgnoresNilHeartbeatPayloadAfterRegister(t *testing.T) {
	agent := &agentdomain.Agent{ID: 63, InstanceID: "agt-63"}
	lifecycle := &runtimeLifecycleStub{}
	svc := NewControlPlaneService(&agentFinderStub{agent: agent}, lifecycle, &taskRuntimeStub{})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-63"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*agentcontrolv1.ConnectRequest{
			runtimeRegisterSessionRequest("agt-63", "session-63"),
			{Payload: &agentcontrolv1.ConnectRequest_Heartbeat{}},
		},
	}

	if err := svc.Connect(stream); err != nil {
		t.Fatalf("expected nil heartbeat payload to be ignored, got %v", err)
	}
	if len(lifecycle.heartbeats) != 1 {
		t.Fatalf("expected only the registration snapshot to be recorded, got %+v", lifecycle.heartbeats)
	}
}

func TestConnectRejectsHeartbeatMappingErrorAfterIdentityValidation(t *testing.T) {
	agent := &agentdomain.Agent{ID: 65, InstanceID: "agt-65"}
	svc := NewControlPlaneService(&agentFinderStub{agent: agent}, &runtimeLifecycleStub{}, &taskRuntimeStub{})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-65"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*agentcontrolv1.ConnectRequest{
			runtimeRegisterSessionRequest("agt-65", "session-65"),
			{
				Payload: &agentcontrolv1.ConnectRequest_Heartbeat{
					Heartbeat: &agentcontrolv1.Heartbeat{
						Agent:                 resourcenames.Agent("agt-65"),
						Session:               resourcenames.AgentSession("agt-65", "session-65"),
						ObservedHostname:      "node-65",
						CompatibilityRevision: "engine-execution-diagnostics-r1",
						Health: &agentcontrolv1.HealthStatus{
							State: agentcontrolv1.HealthState_HEALTH_STATE_UNSPECIFIED,
						},
					},
				},
			},
		},
	}

	err := svc.Connect(stream)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestConnectIgnoresNilTerminalTaskResultPayloadAfterRegister(t *testing.T) {
	agent := &agentdomain.Agent{ID: 64, InstanceID: "agt-64"}
	tasks := &taskRuntimeStub{}
	svc := NewControlPlaneService(&agentFinderStub{agent: agent}, &runtimeLifecycleStub{}, tasks)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-64"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*agentcontrolv1.ConnectRequest{
			runtimeRegisterSessionRequest("agt-64", "session-64"),
			{Payload: &agentcontrolv1.ConnectRequest_TerminalTaskResult{}},
		},
	}

	if err := svc.Connect(stream); err != nil {
		t.Fatalf("expected nil terminal task result payload to be ignored, got %v", err)
	}
	if len(tasks.reportedResults) != 0 {
		t.Fatalf("expected nil terminal task result payload not to report results, got %+v", tasks.reportedResults)
	}
}

func TestConnectRejectsTerminalTaskResultWithMismatchedCompatibilityRevision(t *testing.T) {
	agent := &agentdomain.Agent{ID: 640, InstanceID: "agt-640"}
	tasks := &taskRuntimeStub{}
	svc := NewControlPlaneService(&agentFinderStub{agent: agent}, &runtimeLifecycleStub{}, tasks)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-640"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*agentcontrolv1.ConnectRequest{
			runtimeRegisterSessionRequest(agent.InstanceID, "session-640"),
			{Payload: &agentcontrolv1.ConnectRequest_TerminalTaskResult{TerminalTaskResult: &agentcontrolv1.TerminalTaskResult{
				Task:                  resourcenames.Task(64, 640),
				Result:                agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_SUCCEEDED,
				Diagnostics:           terminalDiagnosticsProtoForTest(),
				CompatibilityRevision: "pre-cut-revision",
			}}},
		},
	}

	err := svc.Connect(stream)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("mismatched terminal revision error = %v", err)
	}
	if len(tasks.reportedResults) != 0 {
		t.Fatalf("mismatched terminal revision reached task bridge: %+v", tasks.reportedResults)
	}
}

func TestConnectRejectsRequestTaskFromStaleSession(t *testing.T) {
	agent := &agentdomain.Agent{ID: 66, InstanceID: "agt-66"}
	svc := NewControlPlaneService(&agentFinderStub{agent: agent}, &runtimeLifecycleStub{}, &taskRuntimeStub{}, NewAgentStreamRegistry())

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-66"))
	first := newBlockingConnectStream(ctx)
	firstDone := make(chan error, 1)
	go func() { firstDone <- svc.Connect(first) }()
	first.recvCh <- runtimeRegisterSessionRequest("agt-66", "session-old")
	if event := first.mustRecvEvent(t); event.GetSessionReady() == nil {
		t.Fatalf("expected first session_ready event, got %+v", event)
	}

	second := newBlockingConnectStream(ctx)
	secondDone := make(chan error, 1)
	go func() { secondDone <- svc.Connect(second) }()
	second.recvCh <- runtimeRegisterSessionRequest("agt-66", "session-new")
	if event := second.mustRecvEvent(t); event.GetSessionReady() == nil {
		t.Fatalf("expected second session_ready event, got %+v", event)
	}

	first.recvCh <- &agentcontrolv1.ConnectRequest{
		Payload: &agentcontrolv1.ConnectRequest_RequestTask{RequestTask: &agentcontrolv1.RequestTask{
			RequestId: "550e8400-e29b-41d4-a716-446655440000",
		}},
	}
	select {
	case err := <-firstDone:
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("expected stale request_task rejection, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for stale request_task rejection")
	}

	second.pushRecvError(errors.New("stop"))
	<-secondDone
}

func TestConnectRejectsTerminalTaskResultFromStaleSession(t *testing.T) {
	agent := &agentdomain.Agent{ID: 67, InstanceID: "agt-67"}
	svc := NewControlPlaneService(&agentFinderStub{agent: agent}, &runtimeLifecycleStub{}, &taskRuntimeStub{}, NewAgentStreamRegistry())

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-67"))
	first := newBlockingConnectStream(ctx)
	firstDone := make(chan error, 1)
	go func() { firstDone <- svc.Connect(first) }()
	first.recvCh <- runtimeRegisterSessionRequest("agt-67", "session-old")
	if event := first.mustRecvEvent(t); event.GetSessionReady() == nil {
		t.Fatalf("expected first session_ready event, got %+v", event)
	}

	second := newBlockingConnectStream(ctx)
	secondDone := make(chan error, 1)
	go func() { secondDone <- svc.Connect(second) }()
	second.recvCh <- runtimeRegisterSessionRequest("agt-67", "session-new")
	if event := second.mustRecvEvent(t); event.GetSessionReady() == nil {
		t.Fatalf("expected second session_ready event, got %+v", event)
	}

	first.recvCh <- &agentcontrolv1.ConnectRequest{
		Payload: &agentcontrolv1.ConnectRequest_TerminalTaskResult{
			TerminalTaskResult: &agentcontrolv1.TerminalTaskResult{Task: resourcenames.Task(1, 1), Result: agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_SUCCEEDED, CompatibilityRevision: testCompatibilityRevision},
		},
	}
	select {
	case err := <-firstDone:
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("expected stale terminal_task_result rejection, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for stale terminal_task_result rejection")
	}

	second.pushRecvError(errors.New("stop"))
	<-secondDone
}

func TestConnectReturnsTaskAssignSendError(t *testing.T) {
	agent := &agentdomain.Agent{ID: 69, InstanceID: "agt-69"}
	svc := NewControlPlaneService(
		&agentFinderStub{agent: agent},
		&runtimeLifecycleStub{},
		&executionClaimRuntimeStub{},
	)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token-69"))
	stream := &scriptedConnectStream{
		ctx: ctx,
		incoming: []*agentcontrolv1.ConnectRequest{
			runtimeRegisterSessionRequest("agt-69", "session-69"),
			{Payload: &agentcontrolv1.ConnectRequest_RequestTask{RequestTask: &agentcontrolv1.RequestTask{
				RequestId: "550e8400-e29b-41d4-a716-446655440000",
			}}},
		},
		sendErrAfter: 2,
	}

	err := svc.Connect(stream)
	if err == nil || err.Error() != "send failed" {
		t.Fatalf("expected send failure from task_assign, got %v", err)
	}
}

func TestForwardOutboundEventsStopsWhenContextCancelled(t *testing.T) {
	svc := &ControlPlaneService{}
	ctx, cancel := context.WithCancel(context.Background())
	stream := newRecordingConnectStream(ctx)
	outbound := make(chan *agentcontrolv1.ConnectResponse)
	done := make(chan struct{})

	go func() {
		svc.forwardOutboundEvents(ctx, &sync.Mutex{}, stream, outbound)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting forwardOutboundEvents to exit on context cancel")
	}
}

func TestForwardOutboundEventsSkipsNilEvent(t *testing.T) {
	svc := &ControlPlaneService{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := newRecordingConnectStream(ctx)
	outbound := make(chan *agentcontrolv1.ConnectResponse, 2)
	done := make(chan struct{})

	go func() {
		svc.forwardOutboundEvents(ctx, &sync.Mutex{}, stream, outbound)
		close(done)
	}()

	outbound <- nil
	outbound <- &agentcontrolv1.ConnectResponse{
		Payload: &agentcontrolv1.ConnectResponse_ConfigUpdate{
			ConfigUpdate: &agentcontrolv1.ConfigUpdate{},
		},
	}

	select {
	case event := <-stream.sentCh:
		if event.GetConfigUpdate() == nil {
			t.Fatalf("unexpected forwarded event: %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting forwarded event after nil event")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting forwardOutboundEvents to exit after nil event")
	}

	if stream.sendCall != 1 {
		t.Fatalf("expected only non-nil event to be sent, got %d sends", stream.sendCall)
	}
}

func TestForwardOutboundEventsStopsOnSendError(t *testing.T) {
	svc := &ControlPlaneService{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := newRecordingConnectStream(ctx)
	stream.sendErr = errors.New("send failed")
	outbound := make(chan *agentcontrolv1.ConnectResponse, 1)
	done := make(chan struct{})

	go func() {
		svc.forwardOutboundEvents(ctx, &sync.Mutex{}, stream, outbound)
		close(done)
	}()

	outbound <- &agentcontrolv1.ConnectResponse{
		Payload: &agentcontrolv1.ConnectResponse_TaskCancel{
			TaskCancel: &agentcontrolv1.TaskCancel{Task: resourcenames.Task(1, 9)},
		},
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting forwardOutboundEvents to exit on send error")
	}

	if stream.sendCall != 1 {
		t.Fatalf("expected one send attempt before exit, got %d", stream.sendCall)
	}
}
