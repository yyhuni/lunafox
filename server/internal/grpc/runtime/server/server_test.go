package server

import (
	"context"
	"errors"
	"testing"
	"time"

	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/runtime/auth"
	"github.com/yyhuni/lunafox/server/internal/grpc/runtime/service"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestNewRejectsNilService(t *testing.T) {
	_, err := New(
		"127.0.0.1:0",
		nil,
		service.NewAgentDataProxyService(),
	)
	if err == nil {
		t.Fatalf("expected error when agent runtime service is nil")
	}
}

func TestServerRegistersRuntimeServices(t *testing.T) {
	srv, err := New(
		"127.0.0.1:0",
		service.NewAgentRuntimeService(),
		service.NewAgentDataProxyService(),
	)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve()
	}()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	connCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(
		connCtx,
		srv.Addr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("dial grpc failed: %v", err)
	}
	defer conn.Close()

	client := runtimev1.NewAgentDataProxyServiceClient(conn)
	_, callErr := client.GetProviderConfig(context.Background(), &runtimev1.GetProviderConfigRequest{})
	if code := status.Code(callErr); code != codes.Unimplemented {
		t.Fatalf("expected unimplemented from skeleton service, got=%s err=%v", code, callErr)
	}
}

func TestServerDataProxyRejectsMissingAgentKey(t *testing.T) {
	srv, err := New(
		"127.0.0.1:0",
		service.NewAgentRuntimeService(),
		service.NewAgentDataProxyServiceWithDeps(
			&providerConfigRuntimeStub{content: "key: value"},
			&wordlistRuntimeStub{},
		).WithAgentFinder(agentFinderStub{agent: &agentdomain.Agent{ID: 101}}),
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

	connCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(
		connCtx,
		srv.Addr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("dial grpc failed: %v", err)
	}
	defer conn.Close()

	client := runtimev1.NewAgentDataProxyServiceClient(conn)
	_, callErr := client.GetProviderConfig(context.Background(), &runtimev1.GetProviderConfigRequest{
		ScanId:   1,
		ToolName: "subfinder",
	})
	if code := status.Code(callErr); code != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got=%s err=%v", code, callErr)
	}
}

func TestServerAgentRuntimeReconnectIntegration(t *testing.T) {
	agent := &agentdomain.Agent{ID: 21, MaxTasks: 2, CPUThreshold: 80, MemThreshold: 80, DiskThreshold: 80}
	registry := service.NewAgentStreamRegistry()
	runtimeSvc := service.NewAgentRuntimeServiceWithDeps(
		agentFinderStub{agent: agent},
		&runtimeLifecycleStub{},
		&taskRuntimeStub{},
		registry,
	)

	srv, err := New(
		"127.0.0.1:0",
		runtimeSvc,
		service.NewAgentDataProxyService(),
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

	connCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(
		connCtx,
		srv.Addr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("dial grpc failed: %v", err)
	}
	defer conn.Close()

	client := runtimev1.NewAgentRuntimeServiceClient(conn)

	// first connection
	firstCtx, firstCancel := context.WithCancel(metadata.AppendToOutgoingContext(context.Background(), grpcauth.AgentKeyHeader, "agent-key"))
	firstStream, err := client.Connect(firstCtx)
	if err != nil {
		t.Fatalf("connect stream #1 failed: %v", err)
	}
	firstEvent, err := firstStream.Recv()
	if err != nil {
		t.Fatalf("recv stream #1 failed: %v", err)
	}
	if firstEvent.GetConfigUpdate() == nil {
		t.Fatalf("expected config update on first stream, got %+v", firstEvent)
	}
	firstCancel()
	for {
		_, recvErr := firstStream.Recv()
		if recvErr != nil {
			break
		}
	}

	// reconnect
	secondCtx, secondCancel := context.WithCancel(metadata.AppendToOutgoingContext(context.Background(), grpcauth.AgentKeyHeader, "agent-key"))
	defer secondCancel()
	secondStream, err := client.Connect(secondCtx)
	if err != nil {
		t.Fatalf("connect stream #2 failed: %v", err)
	}
	secondEvent, err := secondStream.Recv()
	if err != nil {
		t.Fatalf("recv stream #2 failed: %v", err)
	}
	if secondEvent.GetConfigUpdate() == nil {
		t.Fatalf("expected config update on second stream, got %+v", secondEvent)
	}
}

func TestServerAgentRuntimeRejectsInvalidAgentKey(t *testing.T) {
	runtimeSvc := service.NewAgentRuntimeServiceWithDeps(
		agentFinderStub{err: errors.New("invalid")},
		&runtimeLifecycleStub{},
		&taskRuntimeStub{},
	)
	srv, err := New(
		"127.0.0.1:0",
		runtimeSvc,
		service.NewAgentDataProxyService(),
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

	connCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(
		connCtx,
		srv.Addr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("dial grpc failed: %v", err)
	}
	defer conn.Close()

	client := runtimev1.NewAgentRuntimeServiceClient(conn)
	stream, err := client.Connect(metadata.AppendToOutgoingContext(context.Background(), grpcauth.AgentKeyHeader, "bad-key"))
	if err != nil {
		t.Fatalf("connect stream failed: %v", err)
	}
	_, recvErr := stream.Recv()
	if code := status.Code(recvErr); code != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got=%s err=%v", code, recvErr)
	}
}

type agentFinderStub struct {
	agent *agentdomain.Agent
	err   error
}

func (stub agentFinderStub) FindByAPIKey(context.Context, string) (*agentdomain.Agent, error) {
	return stub.agent, stub.err
}

type runtimeLifecycleStub struct{}

func (stub *runtimeLifecycleStub) OnConnected(context.Context, *agentdomain.Agent, string) error {
	return nil
}

func (stub *runtimeLifecycleStub) OnDisconnected(context.Context, int) error {
	return nil
}

func (stub *runtimeLifecycleStub) HandleMessage(context.Context, int, agentapp.RuntimeMessageInput) error {
	return nil
}

type taskRuntimeStub struct{}

func (stub *taskRuntimeStub) PullTask(context.Context, int) (*scanapp.TaskAssignment, error) {
	return nil, nil
}

func (stub *taskRuntimeStub) UpdateStatus(context.Context, int, int, string, *scanapp.FailureDetail) error {
	return nil
}

type providerConfigRuntimeStub struct {
	content string
	err     error
}

func (stub *providerConfigRuntimeStub) GetProviderConfig(_ int, _ string) (string, error) {
	if stub.err != nil {
		return "", stub.err
	}
	return stub.content, nil
}

type wordlistRuntimeStub struct{}

func (wordlistRuntimeStub) GetByName(string) (*catalogapp.Wordlist, error) {
	return nil, errors.New("not implemented")
}

func (wordlistRuntimeStub) GetFilePath(string) (string, error) {
	return "", errors.New("not implemented")
}
