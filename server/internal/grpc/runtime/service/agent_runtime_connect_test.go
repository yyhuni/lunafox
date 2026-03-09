package service

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/runtime/auth"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestConnectRejectsMissingAgentKey(t *testing.T) {
	svc := NewAgentRuntimeServiceWithDeps(
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

func TestConnectHandlesRequestTask(t *testing.T) {
	agent := &agentdomain.Agent{
		ID:            7,
		MaxTasks:      3,
		CPUThreshold:  80,
		MemThreshold:  85,
		DiskThreshold: 90,
	}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	tasks := &taskRuntimeStub{
		pullResult: &scanapp.TaskAssignment{
			TaskID:         11,
			ScanID:         12,
			Stage:          2,
			WorkflowID:     "subdomain_discovery",
			TargetID:       99,
			TargetName:     "example.com",
			TargetType:     "domain",
			WorkspaceDir:   "/opt/lunafox/results/scan_12/task_11",
			WorkflowConfig: map[string]any{"mode": "fast"},
		},
	}

	svc := NewAgentRuntimeServiceWithDeps(finder, lifecycle, tasks)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentKeyHeader, "apikey-1"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*runtimev1.AgentRuntimeRequest{
			{
				Payload: &runtimev1.AgentRuntimeRequest_RequestTask{
					RequestTask: &runtimev1.RequestTask{},
				},
			},
		},
	}

	if err := svc.Connect(stream); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	if finder.lastAPIKey != "apikey-1" {
		t.Fatalf("unexpected api key: %q", finder.lastAPIKey)
	}
	if tasks.pullCalls != 1 {
		t.Fatalf("expected pull called once, got=%d", tasks.pullCalls)
	}
	if !lifecycle.connected || !lifecycle.disconnected {
		t.Fatalf("expected connected/disconnected lifecycle callbacks")
	}
	if len(stream.sent) != 2 {
		t.Fatalf("expected 2 events (config_update + task_assign), got=%d", len(stream.sent))
	}

	assign := stream.sent[1].GetTaskAssign()
	if assign == nil || !assign.Found || assign.TaskId != 11 {
		t.Fatalf("unexpected task assign event: %+v", stream.sent[1])
	}
}

func TestConnectUsesPeerIPAddressForLifecycle(t *testing.T) {
	agent := &agentdomain.Agent{ID: 19}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	svc := NewAgentRuntimeServiceWithDeps(finder, lifecycle, &taskRuntimeStub{})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentKeyHeader, "apikey-19"))
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: &net.TCPAddr{
			IP:   net.ParseIP("203.0.113.19"),
			Port: 50001,
		},
	})
	stream := &fakeConnectStream{ctx: ctx}

	if err := svc.Connect(stream); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	if lifecycle.connectedIP != "203.0.113.19" {
		t.Fatalf("expected peer ip 203.0.113.19, got %q", lifecycle.connectedIP)
	}
}

func TestConnectHandlesHeartbeatAndTaskStatus(t *testing.T) {
	agent := &agentdomain.Agent{ID: 9}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	tasks := &taskRuntimeStub{}
	svc := NewAgentRuntimeServiceWithDeps(finder, lifecycle, tasks)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentKeyHeader, "apikey-9"))
	stream := &fakeConnectStream{
		ctx: ctx,
		incoming: []*runtimev1.AgentRuntimeRequest{
			{
				Payload: &runtimev1.AgentRuntimeRequest_Heartbeat{
					Heartbeat: &runtimev1.Heartbeat{
						CpuUsage:      21.5,
						MemUsage:      35.2,
						DiskUsage:     41.3,
						RunningTasks:  2,
						AgentVersion:  "v1.2.3",
						Hostname:      "agent-1",
						UptimeSeconds: 1234,
					},
				},
			},
			{
				Payload: &runtimev1.AgentRuntimeRequest_TaskStatus{
					TaskStatus: &runtimev1.TaskStatus{
						TaskId:      77,
						Status:      "failed",
						Message:     "boom",
						FailureKind: "runtime_error",
					},
				},
			},
		},
	}

	if err := svc.Connect(stream); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	if len(lifecycle.messages) != 1 {
		t.Fatalf("expected heartbeat message delegated once, got=%d", len(lifecycle.messages))
	}
	if lifecycle.messages[0].Type != agentapp.RuntimeMessageTypeHeartbeat {
		t.Fatalf("unexpected runtime message type: %q", lifecycle.messages[0].Type)
	}
	if len(tasks.updates) != 1 {
		t.Fatalf("expected task status update once, got=%d", len(tasks.updates))
	}
	if tasks.updates[0].taskID != 77 || tasks.updates[0].status != "failed" {
		t.Fatalf("unexpected task status update payload: %+v", tasks.updates[0])
	}
	if tasks.updates[0].failureKind != "runtime_error" {
		t.Fatalf("unexpected failure kind: %+v", tasks.updates[0])
	}
}

func TestConnectReceivesPublishedDownlinkEvents(t *testing.T) {
	agent := &agentdomain.Agent{ID: 33}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	tasks := &taskRuntimeStub{}
	registry := NewAgentStreamRegistry()
	svc := NewAgentRuntimeServiceWithDeps(finder, lifecycle, tasks, registry)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentKeyHeader, "apikey-33"))
	stream := newBlockingConnectStream(ctx)
	done := make(chan error, 1)
	go func() {
		done <- svc.Connect(stream)
	}()

	first := stream.mustRecvEvent(t)
	if first.GetConfigUpdate() == nil {
		t.Fatalf("expected first event to be config_update, got %+v", first)
	}

	publisher := NewAgentRuntimeEventPublisher(registry)
	publisher.SendTaskCancel(33, 88)
	cancelEvent := stream.mustRecvEvent(t)
	if cancelEvent.GetTaskCancel() == nil || cancelEvent.GetTaskCancel().GetTaskId() != 88 {
		t.Fatalf("unexpected task cancel event: %+v", cancelEvent)
	}

	delivered := publisher.SendUpdateRequired(33, agentdomain.UpdateRequiredPayload{
		AgentVersion:   "v2.1.0",
		AgentImageRef:  "registry.example.com/lunafox-agent:v2.1.0",
		WorkerImageRef: "registry.example.com/lunafox-worker:v2.1.0",
		WorkerVersion:  "2.1.0",
	})
	if !delivered {
		t.Fatalf("expected update_required event delivery")
	}
	updateEvent := stream.mustRecvEvent(t)
	if updateEvent.GetUpdateRequired() == nil || updateEvent.GetUpdateRequired().GetAgentVersion() != "v2.1.0" {
		t.Fatalf("unexpected update required event: %+v", updateEvent)
	}
	if updateEvent.GetUpdateRequired().GetAgentImageRef() != "registry.example.com/lunafox-agent:v2.1.0" {
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

func TestConnectSupportsReconnect(t *testing.T) {
	agent := &agentdomain.Agent{ID: 44}
	finder := &agentFinderStub{agent: agent}
	lifecycle := &runtimeLifecycleStub{}
	tasks := &taskRuntimeStub{}
	svc := NewAgentRuntimeServiceWithDeps(finder, lifecycle, tasks)

	for i := 0; i < 2; i++ {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentKeyHeader, "apikey-44"))
		stream := &fakeConnectStream{ctx: ctx}
		if err := svc.Connect(stream); err != nil {
			t.Fatalf("connect #%d failed: %v", i+1, err)
		}
	}

	if lifecycle.connectedCount != 2 {
		t.Fatalf("expected 2 connects, got=%d", lifecycle.connectedCount)
	}
	if lifecycle.disconnectedCount != 2 {
		t.Fatalf("expected 2 disconnects, got=%d", lifecycle.disconnectedCount)
	}
}

func TestConnectMapsInternalErrors(t *testing.T) {
	t.Run("on connected failure", func(t *testing.T) {
		svc := NewAgentRuntimeServiceWithDeps(
			&agentFinderStub{agent: &agentdomain.Agent{ID: 1}},
			&runtimeLifecycleStub{onConnectedErr: errors.New("connect failed")},
			&taskRuntimeStub{},
		)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentKeyHeader, "apikey-1"))
		err := svc.Connect(&fakeConnectStream{ctx: ctx})
		if status.Code(err) != codes.Internal {
			t.Fatalf("expected internal, got=%v", err)
		}
	})

	t.Run("heartbeat delegation failure", func(t *testing.T) {
		svc := NewAgentRuntimeServiceWithDeps(
			&agentFinderStub{agent: &agentdomain.Agent{ID: 2}},
			&runtimeLifecycleStub{handleErr: errors.New("heartbeat failed")},
			&taskRuntimeStub{},
		)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentKeyHeader, "apikey-2"))
		stream := &fakeConnectStream{
			ctx: ctx,
			incoming: []*runtimev1.AgentRuntimeRequest{
				{
					Payload: &runtimev1.AgentRuntimeRequest_Heartbeat{
						Heartbeat: &runtimev1.Heartbeat{CpuUsage: 10},
					},
				},
			},
		}
		err := svc.Connect(stream)
		if status.Code(err) != codes.Internal {
			t.Fatalf("expected internal, got=%v", err)
		}
	})

	t.Run("pull task failure", func(t *testing.T) {
		svc := NewAgentRuntimeServiceWithDeps(
			&agentFinderStub{agent: &agentdomain.Agent{ID: 3}},
			&runtimeLifecycleStub{},
			&taskRuntimeStub{pullErr: errors.New("pull failed")},
		)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentKeyHeader, "apikey-3"))
		stream := &fakeConnectStream{
			ctx: ctx,
			incoming: []*runtimev1.AgentRuntimeRequest{
				{Payload: &runtimev1.AgentRuntimeRequest_RequestTask{RequestTask: &runtimev1.RequestTask{}}},
			},
		}
		err := svc.Connect(stream)
		if status.Code(err) != codes.Internal {
			t.Fatalf("expected internal, got=%v", err)
		}
	})

	t.Run("pull task schema invalid failure", func(t *testing.T) {
		svc := NewAgentRuntimeServiceWithDeps(
			&agentFinderStub{agent: &agentdomain.Agent{ID: 34}},
			&runtimeLifecycleStub{},
			&taskRuntimeStub{
				pullErr: scanapp.NewWorkflowError(
					scanapp.WorkflowErrorCodeSchemaInvalid,
					scanapp.WorkflowErrorStageServerSchemaGate,
					"workflow",
					"missing workflow",
					nil,
				),
			},
		)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentKeyHeader, "apikey-34"))
		stream := &fakeConnectStream{
			ctx: ctx,
			incoming: []*runtimev1.AgentRuntimeRequest{
				{Payload: &runtimev1.AgentRuntimeRequest_RequestTask{RequestTask: &runtimev1.RequestTask{}}},
			},
		}
		err := svc.Connect(stream)
		if err != nil {
			t.Fatalf("expected stream keepalive for workflow error, got=%v", err)
		}
		if len(stream.sent) != 2 {
			t.Fatalf("expected config_update + empty task_assign, got=%d", len(stream.sent))
		}
		assign := stream.sent[1].GetTaskAssign()
		if assign == nil || assign.GetFound() {
			t.Fatalf("expected empty task assign event, got=%+v", stream.sent[1])
		}
	})

	t.Run("pull task workflow config invalid failure", func(t *testing.T) {
		svc := NewAgentRuntimeServiceWithDeps(
			&agentFinderStub{agent: &agentdomain.Agent{ID: 35}},
			&runtimeLifecycleStub{},
			&taskRuntimeStub{
				pullErr: scanapp.NewWorkflowError(
					scanapp.WorkflowErrorCodeWorkflowConfigInvalid,
					scanapp.WorkflowErrorStageWorkerValidate,
					"recon",
					"recon stage enabled requires at least one tool enabled",
					nil,
				),
			},
		)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentKeyHeader, "apikey-35"))
		stream := &fakeConnectStream{
			ctx: ctx,
			incoming: []*runtimev1.AgentRuntimeRequest{
				{Payload: &runtimev1.AgentRuntimeRequest_RequestTask{RequestTask: &runtimev1.RequestTask{}}},
			},
		}
		err := svc.Connect(stream)
		if err != nil {
			t.Fatalf("expected stream keepalive for workflow error, got=%v", err)
		}
		if len(stream.sent) != 2 {
			t.Fatalf("expected config_update + empty task_assign, got=%d", len(stream.sent))
		}
		assign := stream.sent[1].GetTaskAssign()
		if assign == nil || assign.GetFound() {
			t.Fatalf("expected empty task assign event, got=%+v", stream.sent[1])
		}
	})

	t.Run("pull task workflow prereq missing failure", func(t *testing.T) {
		svc := NewAgentRuntimeServiceWithDeps(
			&agentFinderStub{agent: &agentdomain.Agent{ID: 36}},
			&runtimeLifecycleStub{},
			&taskRuntimeStub{
				pullErr: scanapp.NewWorkflowError(
					scanapp.WorkflowErrorCodeWorkflowPrereqMissing,
					scanapp.WorkflowErrorStageWorkerPrereq,
					"subfinder",
					"subfinder binary not found",
					nil,
				),
			},
		)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentKeyHeader, "apikey-36"))
		stream := &fakeConnectStream{
			ctx: ctx,
			incoming: []*runtimev1.AgentRuntimeRequest{
				{Payload: &runtimev1.AgentRuntimeRequest_RequestTask{RequestTask: &runtimev1.RequestTask{}}},
			},
		}
		err := svc.Connect(stream)
		if err != nil {
			t.Fatalf("expected stream keepalive for workflow error, got=%v", err)
		}
		if len(stream.sent) != 2 {
			t.Fatalf("expected config_update + empty task_assign, got=%d", len(stream.sent))
		}
		assign := stream.sent[1].GetTaskAssign()
		if assign == nil || assign.GetFound() {
			t.Fatalf("expected empty task assign event, got=%+v", stream.sent[1])
		}
	})

	t.Run("pull task schema invalid format failure", func(t *testing.T) {
		svc := NewAgentRuntimeServiceWithDeps(
			&agentFinderStub{agent: &agentdomain.Agent{ID: 37}},
			&runtimeLifecycleStub{},
			&taskRuntimeStub{
				pullErr: scanapp.NewWorkflowError(
					scanapp.WorkflowErrorCodeSchemaInvalid,
					scanapp.WorkflowErrorStageServerSchemaGate,
					"workflow",
					"workflow must be a non-empty string",
					nil,
				),
			},
		)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentKeyHeader, "apikey-37"))
		stream := &fakeConnectStream{
			ctx: ctx,
			incoming: []*runtimev1.AgentRuntimeRequest{
				{Payload: &runtimev1.AgentRuntimeRequest_RequestTask{RequestTask: &runtimev1.RequestTask{}}},
			},
		}
		err := svc.Connect(stream)
		if err != nil {
			t.Fatalf("expected stream keepalive for workflow error, got=%v", err)
		}
		if len(stream.sent) != 2 {
			t.Fatalf("expected config_update + empty task_assign, got=%d", len(stream.sent))
		}
		assign := stream.sent[1].GetTaskAssign()
		if assign == nil || assign.GetFound() {
			t.Fatalf("expected empty task assign event, got=%+v", stream.sent[1])
		}
	})

	t.Run("pull task schema invalid enum failure", func(t *testing.T) {
		svc := NewAgentRuntimeServiceWithDeps(
			&agentFinderStub{agent: &agentdomain.Agent{ID: 38}},
			&runtimeLifecycleStub{},
			&taskRuntimeStub{
				pullErr: scanapp.NewWorkflowError(
					scanapp.WorkflowErrorCodeSchemaInvalid,
					scanapp.WorkflowErrorStageServerSchemaGate,
					"subdomain_discovery",
					"workflow subdomain_discovery is not supported by current schema",
					nil,
				),
			},
		)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentKeyHeader, "apikey-38"))
		stream := &fakeConnectStream{
			ctx: ctx,
			incoming: []*runtimev1.AgentRuntimeRequest{
				{Payload: &runtimev1.AgentRuntimeRequest_RequestTask{RequestTask: &runtimev1.RequestTask{}}},
			},
		}
		err := svc.Connect(stream)
		if err != nil {
			t.Fatalf("expected stream keepalive for workflow error, got=%v", err)
		}
		if len(stream.sent) != 2 {
			t.Fatalf("expected config_update + empty task_assign, got=%d", len(stream.sent))
		}
		assign := stream.sent[1].GetTaskAssign()
		if assign == nil || assign.GetFound() {
			t.Fatalf("expected empty task assign event, got=%+v", stream.sent[1])
		}
	})

	t.Run("task status update failure", func(t *testing.T) {
		svc := NewAgentRuntimeServiceWithDeps(
			&agentFinderStub{agent: &agentdomain.Agent{ID: 4}},
			&runtimeLifecycleStub{},
			&taskRuntimeStub{updateErr: errors.New("update failed")},
		)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentKeyHeader, "apikey-4"))
		stream := &fakeConnectStream{
			ctx: ctx,
			incoming: []*runtimev1.AgentRuntimeRequest{
				{
					Payload: &runtimev1.AgentRuntimeRequest_TaskStatus{
						TaskStatus: &runtimev1.TaskStatus{
							TaskId: 1,
							Status: "failed",
						},
					},
				},
			},
		}
		err := svc.Connect(stream)
		if status.Code(err) != codes.Internal {
			t.Fatalf("expected internal, got=%v", err)
		}
	})
}

type fakeConnectStream struct {
	ctx      context.Context
	incoming []*runtimev1.AgentRuntimeRequest
	sent     []*runtimev1.AgentRuntimeEvent
}

func (s *fakeConnectStream) Context() context.Context { return s.ctx }

func (s *fakeConnectStream) Send(msg *runtimev1.AgentRuntimeEvent) error {
	s.sent = append(s.sent, msg)
	return nil
}

func (s *fakeConnectStream) Recv() (*runtimev1.AgentRuntimeRequest, error) {
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

type blockingConnectStream struct {
	ctx     context.Context
	recvCh  chan *runtimev1.AgentRuntimeRequest
	recvErr chan error
	sentCh  chan *runtimev1.AgentRuntimeEvent
	mu      sync.Mutex
	sent    []*runtimev1.AgentRuntimeEvent
}

func newBlockingConnectStream(ctx context.Context) *blockingConnectStream {
	return &blockingConnectStream{
		ctx:     ctx,
		recvCh:  make(chan *runtimev1.AgentRuntimeRequest),
		recvErr: make(chan error, 1),
		sentCh:  make(chan *runtimev1.AgentRuntimeEvent, 16),
	}
}

func (s *blockingConnectStream) Context() context.Context { return s.ctx }

func (s *blockingConnectStream) Send(event *runtimev1.AgentRuntimeEvent) error {
	s.mu.Lock()
	s.sent = append(s.sent, event)
	s.mu.Unlock()
	s.sentCh <- event
	return nil
}

func (s *blockingConnectStream) Recv() (*runtimev1.AgentRuntimeRequest, error) {
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

func (s *blockingConnectStream) mustRecvEvent(t *testing.T) *runtimev1.AgentRuntimeEvent {
	t.Helper()
	select {
	case event := <-s.sentCh:
		return event
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for outbound runtime event")
		return nil
	}
}

func (s *blockingConnectStream) SetHeader(metadata.MD) error  { return nil }
func (s *blockingConnectStream) SendHeader(metadata.MD) error { return nil }
func (s *blockingConnectStream) SetTrailer(metadata.MD)       {}
func (s *blockingConnectStream) SendMsg(any) error            { return nil }
func (s *blockingConnectStream) RecvMsg(any) error            { return nil }

type agentFinderStub struct {
	agent      *agentdomain.Agent
	err        error
	lastAPIKey string
}

func (s *agentFinderStub) FindByAPIKey(_ context.Context, apiKey string) (*agentdomain.Agent, error) {
	s.lastAPIKey = apiKey
	return s.agent, s.err
}

type runtimeLifecycleStub struct {
	connected         bool
	disconnected      bool
	messages          []agentapp.RuntimeMessageInput
	connectedCount    int
	disconnectedCount int
	connectedIP       string
	onConnectedErr    error
	onDisconnectedErr error
	handleErr         error
}

func (s *runtimeLifecycleStub) OnConnected(_ context.Context, _ *agentdomain.Agent, ipAddress string) error {
	if s.onConnectedErr != nil {
		return s.onConnectedErr
	}
	s.connected = true
	s.connectedCount++
	s.connectedIP = ipAddress
	return nil
}

func (s *runtimeLifecycleStub) OnDisconnected(context.Context, int) error {
	if s.onDisconnectedErr != nil {
		return s.onDisconnectedErr
	}
	s.disconnected = true
	s.disconnectedCount++
	return nil
}

func (s *runtimeLifecycleStub) HandleMessage(_ context.Context, _ int, message agentapp.RuntimeMessageInput) error {
	if s.handleErr != nil {
		return s.handleErr
	}
	s.messages = append(s.messages, message)
	return nil
}

type taskUpdate struct {
	agentID     int
	taskID      int
	status      string
	message     string
	failureKind string
}

type taskRuntimeStub struct {
	pullResult *scanapp.TaskAssignment
	pullErr    error
	pullCalls  int
	updateErr  error
	updates    []taskUpdate
}

func (s *taskRuntimeStub) PullTask(context.Context, int) (*scanapp.TaskAssignment, error) {
	s.pullCalls++
	return s.pullResult, s.pullErr
}

func (s *taskRuntimeStub) UpdateStatus(_ context.Context, agentID, taskID int, status, errorMessage, failureKind string) error {
	s.updates = append(s.updates, taskUpdate{agentID: agentID, taskID: taskID, status: status, message: errorMessage, failureKind: failureKind})
	return s.updateErr
}
