package agentserver

import (
	"context"
	"errors"
	"testing"
	"time"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	agentcontrol "github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"
	agentdata "github.com/yyhuni/lunafox/server/internal/grpc/agentdata"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestNewRejectsNilService(t *testing.T) {
	_, err := New(
		"127.0.0.1:0",
		nil,
		&agentdata.DataPlaneService{},
		&agentdata.ExecutionArtifactService{},
	)
	if err == nil {
		t.Fatalf("expected error when agent control-plane service is nil")
	}
}

func TestNewRejectsNilDataPlaneService(t *testing.T) {
	_, err := New(
		"127.0.0.1:0",
		&agentcontrol.ControlPlaneService{},
		nil,
		&agentdata.ExecutionArtifactService{},
	)
	if err == nil {
		t.Fatalf("expected error when agent data-plane service is nil")
	}
}

func TestNewRejectsNilExecutionArtifactService(t *testing.T) {
	_, err := New(
		"127.0.0.1:0",
		&agentcontrol.ControlPlaneService{},
		&agentdata.DataPlaneService{},
		nil,
	)
	if err == nil {
		t.Fatal("expected error when execution artifact service is nil")
	}
}

func TestServerRegistersRuntimeServices(t *testing.T) {
	srv, err := New(
		"127.0.0.1:0",
		&agentcontrol.ControlPlaneService{},
		&agentdata.DataPlaneService{},
		&agentdata.ExecutionArtifactService{},
	)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}
	if _, ok := srv.grpcServer.GetServiceInfo()[agentdatav1.ExecutionArtifactService_ServiceDesc.ServiceName]; !ok {
		t.Fatal("execution artifact service is not registered")
	}
	if _, ok := srv.grpcServer.GetServiceInfo()[agentdatav1.DataPlaneService_ServiceDesc.ServiceName]; !ok {
		t.Fatal("Agent reporting data-plane service is not registered")
	}
	if _, ok := srv.grpcServer.GetServiceInfo()[agentcontrolv1.ControlPlaneService_ServiceDesc.ServiceName]; !ok {
		t.Fatal("Agent control-plane service is not registered")
	}
	if got := len(srv.grpcServer.GetServiceInfo()); got != 3 {
		t.Fatalf("registered runtime services = %d, want exactly one registration for each of three services", got)
	}
	if srv.listener == nil || srv.addr == "" {
		t.Fatal("Agent runtime services must share the one initialized listener")
	}
}

func TestServerAgentDataPlaneRejectsMissingAgentAuthenticationToken(t *testing.T) {
	srv, err := New(
		"127.0.0.1:0",
		&agentcontrol.ControlPlaneService{},
		agentdata.NewDataPlaneService(
			agentFinderStub{agent: &agentdomain.Agent{ID: 101}},
			agentdata.ResultIngestDataPlanes{},
		),

		&agentdata.ExecutionArtifactService{},
	)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	go func() {
		_ = srv.Serve()
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	conn, err := dialRuntimeServer(srv.Addr())
	if err != nil {
		t.Fatalf("dial grpc failed: %v", err)
	}
	defer conn.Close()

	client := agentdatav1.NewDataPlaneServiceClient(conn)
	_, callErr := client.BatchIngestTaskResults(context.Background(), &agentdatav1.BatchIngestTaskResultsRequest{})
	if code := status.Code(callErr); code != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got=%s err=%v", code, callErr)
	}
}

func TestServerAgentControlReconnectIntegration(t *testing.T) {
	agent := &agentdomain.Agent{ID: 21, InstanceID: "agt-21", MaxTasks: 2, CPUThreshold: 80, MemThreshold: 80, DiskThreshold: 80}
	registry := agentcontrol.NewAgentStreamRegistry()
	runtimeSvc := agentcontrol.NewControlPlaneService(
		agentFinderStub{agent: agent},
		&runtimeLifecycleStub{},
		&taskRuntimeStub{},
		registry,
	)

	srv, err := New(
		"127.0.0.1:0",
		runtimeSvc,
		&agentdata.DataPlaneService{},
		&agentdata.ExecutionArtifactService{},
	)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	go func() {
		_ = srv.Serve()
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	conn, err := dialRuntimeServer(srv.Addr())
	if err != nil {
		t.Fatalf("dial grpc failed: %v", err)
	}
	defer conn.Close()

	client := agentcontrolv1.NewControlPlaneServiceClient(conn)

	// first connection
	firstCtx, firstCancel := context.WithCancel(metadata.AppendToOutgoingContext(context.Background(), grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token"))
	firstStream, err := client.Connect(firstCtx)
	if err != nil {
		t.Fatalf("connect stream #1 failed: %v", err)
	}
	if err := firstStream.Send(&agentcontrolv1.ConnectRequest{
		Payload: &agentcontrolv1.ConnectRequest_RegisterSession{
			RegisterSession: &agentcontrolv1.RegisterSession{
				Agent:                 resourcenames.Agent("agt-21"),
				Session:               resourcenames.AgentSession("agt-21", "session-1"),
				ObservedHostname:      "node-1",
				AgentVersion:          "v1.0.0",
				CompatibilityRevision: "engine-execution-diagnostics-r1",
			},
		},
	}); err != nil {
		t.Fatalf("register session #1 failed: %v", err)
	}
	firstEvent, err := firstStream.Recv()
	if err != nil {
		t.Fatalf("recv stream #1 failed: %v", err)
	}
	if firstEvent.GetSessionReady() == nil {
		t.Fatalf("expected session_ready on first stream, got %+v", firstEvent)
	}
	firstEpoch := firstEvent.GetSessionReady().GetSessionEpoch()
	if firstEpoch <= 0 {
		t.Fatalf("expected positive first session epoch, got %d", firstEpoch)
	}
	firstCancel()
	for {
		_, recvErr := firstStream.Recv()
		if recvErr != nil {
			break
		}
	}

	// reconnect
	secondCtx, secondCancel := context.WithCancel(metadata.AppendToOutgoingContext(context.Background(), grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token"))
	defer secondCancel()
	secondStream, err := client.Connect(secondCtx)
	if err != nil {
		t.Fatalf("connect stream #2 failed: %v", err)
	}
	if err := secondStream.Send(&agentcontrolv1.ConnectRequest{
		Payload: &agentcontrolv1.ConnectRequest_RegisterSession{
			RegisterSession: &agentcontrolv1.RegisterSession{
				Agent:                 resourcenames.Agent("agt-21"),
				Session:               resourcenames.AgentSession("agt-21", "session-1"),
				ObservedHostname:      "node-2",
				AgentVersion:          "v1.0.0",
				CompatibilityRevision: "engine-execution-diagnostics-r1",
			},
		},
	}); err != nil {
		t.Fatalf("register session #2 failed: %v", err)
	}
	secondEvent, err := secondStream.Recv()
	if err != nil {
		t.Fatalf("recv stream #2 failed: %v", err)
	}
	if secondEvent.GetSessionReady() == nil {
		t.Fatalf("expected session_ready on second stream, got %+v", secondEvent)
	}
	secondEpoch := secondEvent.GetSessionReady().GetSessionEpoch()
	if secondEpoch != firstEpoch {
		t.Fatalf("expected same-session reconnect to reuse epoch, first=%d second=%d", firstEpoch, secondEpoch)
	}
}

func TestServerAgentControlRejectsInvalidAgentAuthenticationToken(t *testing.T) {
	runtimeSvc := agentcontrol.NewControlPlaneService(
		agentFinderStub{err: errors.New("invalid")},
		&runtimeLifecycleStub{},
		&taskRuntimeStub{},
	)
	srv, err := New(
		"127.0.0.1:0",
		runtimeSvc,
		&agentdata.DataPlaneService{},
		&agentdata.ExecutionArtifactService{},
	)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}
	go func() {
		_ = srv.Serve()
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	conn, err := dialRuntimeServer(srv.Addr())
	if err != nil {
		t.Fatalf("dial grpc failed: %v", err)
	}
	defer conn.Close()

	client := agentcontrolv1.NewControlPlaneServiceClient(conn)
	stream, err := client.Connect(metadata.AppendToOutgoingContext(context.Background(), grpcauth.AgentAuthenticationTokenMetadataKey, "bad-token"))
	if err != nil {
		t.Fatalf("connect stream failed: %v", err)
	}
	_, recvErr := stream.Recv()
	if code := status.Code(recvErr); code != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got=%s err=%v", code, recvErr)
	}
}

func TestServeRejectsUninitializedServer(t *testing.T) {
	var srv Server

	err := srv.Serve()
	if err == nil || err.Error() != "agent plane grpc server not initialized" {
		t.Fatalf("expected uninitialized serve error, got %v", err)
	}
}

func TestShutdownNilServerNoop(t *testing.T) {
	var srv *Server

	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("expected nil server shutdown to be noop, got %v", err)
	}
}

func TestShutdownReturnsContextErrorWhenActiveStreamsPreventGracefulStop(t *testing.T) {
	agent := &agentdomain.Agent{ID: 88, InstanceID: "agt-88"}
	runtimeSvc := agentcontrol.NewControlPlaneService(
		&agentFinderStub{agent: agent},
		&runtimeLifecycleStub{},
		&taskRuntimeStub{},
	)

	srv, err := New(
		"127.0.0.1:0",
		runtimeSvc,
		&agentdata.DataPlaneService{},
		&agentdata.ExecutionArtifactService{},
	)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve()
	}()

	conn, err := dialRuntimeServer(srv.Addr())
	if err != nil {
		t.Fatalf("dial grpc failed: %v", err)
	}
	defer conn.Close()

	client := agentcontrolv1.NewControlPlaneServiceClient(conn)
	stream, err := client.Connect(metadata.AppendToOutgoingContext(context.Background(), grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token"))
	if err != nil {
		t.Fatalf("connect stream failed: %v", err)
	}
	if err := stream.Send(&agentcontrolv1.ConnectRequest{
		Payload: &agentcontrolv1.ConnectRequest_RegisterSession{
			RegisterSession: &agentcontrolv1.RegisterSession{
				Agent:                 resourcenames.Agent("agt-88"),
				Session:               resourcenames.AgentSession("agt-88", "session-88"),
				ObservedHostname:      "node-88",
				AgentVersion:          "v1.0.0",
				CompatibilityRevision: "engine-execution-diagnostics-r1",
			},
		},
	}); err != nil {
		t.Fatalf("register session failed: %v", err)
	}
	event, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv session ready failed: %v", err)
	}
	if event.GetSessionReady() == nil {
		t.Fatalf("expected session_ready event, got %+v", event)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err = srv.Shutdown(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected shutdown deadline exceeded, got %v", err)
	}

	select {
	case <-serveErr:
	case <-time.After(2 * time.Second):
		t.Fatal("expected server serve loop to stop after forced shutdown")
	}
}

func dialRuntimeServer(addr string) (*grpc.ClientConn, error) {
	dialCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	conn.Connect()
	for {
		state := conn.GetState()
		switch state {
		case connectivity.Ready:
			return conn, nil
		case connectivity.Shutdown:
			_ = conn.Close()
			if ctxErr := dialCtx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			return nil, errors.New("runtime server test connection shutdown")
		}
		if !conn.WaitForStateChange(dialCtx, state) {
			_ = conn.Close()
			if ctxErr := dialCtx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			return nil, errors.New("runtime server test connection timed out")
		}
	}
}

type agentFinderStub struct {
	agent *agentdomain.Agent
	err   error
}

func (stub agentFinderStub) FindByAuthenticationToken(context.Context, string) (*agentdomain.Agent, error) {
	return stub.agent, stub.err
}

type runtimeLifecycleStub struct{}

func (stub *runtimeLifecycleStub) OnConnected(context.Context, *agentdomain.Agent, string) error {
	return nil
}

func (stub *runtimeLifecycleStub) OnControlConnectionDetached(context.Context, int) error {
	return nil
}

func (stub *runtimeLifecycleStub) OnDisconnected(context.Context, int) error {
	return nil
}

func (stub *runtimeLifecycleStub) RecordHeartbeat(context.Context, int, agentdomain.AgentHeartbeatEvent) error {
	return nil
}

type taskRuntimeStub struct{}

func (stub *taskRuntimeStub) ClaimNextExecutionPlan(context.Context, int, string, int64, string, agentdomain.AgentExecutionCapabilitySnapshot) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	return nil, nil
}

func (stub *taskRuntimeStub) ReportTerminalTaskResult(context.Context, int, string, int64, int, string, *scanapp.FailureDetail) error {
	return nil
}

func (stub *taskRuntimeStub) FenceSupersededAgentSession(context.Context, int, int64) error {
	return nil
}
