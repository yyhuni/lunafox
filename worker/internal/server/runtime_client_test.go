package server

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	grpcauth "github.com/yyhuni/lunafox/worker/internal/grpc/runtime/auth"
	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type workerRuntimeServiceStub struct {
	runtimev1.UnimplementedWorkerRuntimeServiceServer

	mu sync.Mutex

	lastProviderReq *runtimev1.GetProviderConfigRequest
	lastEnsureReq   *runtimev1.EnsureWordlistRequest
	lastBatchReq    *runtimev1.PostBatchRequest
	lastToken       string
	providerCalls   int
	providerErrs    []error
}

func (s *workerRuntimeServiceStub) GetProviderConfig(ctx context.Context, req *runtimev1.GetProviderConfigRequest) (*runtimev1.GetProviderConfigResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastProviderReq = req
	s.lastToken = readTaskTokenFromContext(ctx)
	s.providerCalls++
	if len(s.providerErrs) > 0 {
		err := s.providerErrs[0]
		s.providerErrs = s.providerErrs[1:]
		if err != nil {
			return nil, err
		}
	}
	return &runtimev1.GetProviderConfigResponse{Content: "key: value"}, nil
}

func (s *workerRuntimeServiceStub) EnsureWordlist(ctx context.Context, req *runtimev1.EnsureWordlistRequest) (*runtimev1.EnsureWordlistResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastEnsureReq = req
	s.lastToken = readTaskTokenFromContext(ctx)
	return &runtimev1.EnsureWordlistResponse{
		LocalPath: filepath.Join(req.LocalBasePath, req.WordlistName),
		FileHash:  "abc123",
	}, nil
}

func (s *workerRuntimeServiceStub) PostBatch(ctx context.Context, req *runtimev1.PostBatchRequest) (*runtimev1.PostBatchResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastBatchReq = req
	s.lastToken = readTaskTokenFromContext(ctx)
	return &runtimev1.PostBatchResponse{Sent: int32(len(req.ItemsJson))}, nil
}

func readTaskTokenFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(grpcauth.TaskTokenHeader)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func TestRuntimeClientIntegration(t *testing.T) {
	stub := &workerRuntimeServiceStub{}
	socketPath := startWorkerRuntimeStubServer(t, stub)

	client, err := NewRuntimeClient(socketPath, "task-token-1", 7001)
	if err != nil {
		t.Fatalf("new runtime client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	cfg, err := client.GetProviderConfig(context.Background(), 11, "subfinder")
	if err != nil {
		t.Fatalf("get provider config: %v", err)
	}
	if cfg.Content != "key: value" {
		t.Fatalf("unexpected provider config content: %q", cfg.Content)
	}

	wordlistPath, err := client.EnsureWordlistLocal(context.Background(), "subs.txt", "/opt/lunafox/wordlists")
	if err != nil {
		t.Fatalf("ensure wordlist local: %v", err)
	}
	if wordlistPath != "/opt/lunafox/wordlists/subs.txt" {
		t.Fatalf("unexpected wordlist path: %q", wordlistPath)
	}

	if err := client.PostBatch(context.Background(), 11, 22, "subdomain", []any{
		map[string]any{"name": "api.example.com"},
		map[string]any{"name": "www.example.com"},
	}); err != nil {
		t.Fatalf("post batch: %v", err)
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.lastProviderReq == nil || stub.lastProviderReq.GetTaskId() != 7001 {
		t.Fatalf("unexpected provider request: %+v", stub.lastProviderReq)
	}
	if stub.lastEnsureReq == nil || stub.lastEnsureReq.GetTaskId() != 7001 {
		t.Fatalf("unexpected ensure request: %+v", stub.lastEnsureReq)
	}
	if stub.lastBatchReq == nil || stub.lastBatchReq.GetTaskId() != 7001 {
		t.Fatalf("unexpected batch request: %+v", stub.lastBatchReq)
	}
	if stub.lastBatchReq.GetDataType() != "subdomain" || len(stub.lastBatchReq.GetItemsJson()) != 2 {
		t.Fatalf("unexpected batch payload: %+v", stub.lastBatchReq)
	}
	if stub.lastToken != "task-token-1" {
		t.Fatalf("unexpected task token metadata: %q", stub.lastToken)
	}
}

func TestNewRuntimeClientValidation(t *testing.T) {
	if _, err := NewRuntimeClient("", "token", 1); err == nil {
		t.Fatalf("expected empty socket path to fail")
	}
	if _, err := NewRuntimeClient("/tmp/worker.sock", "", 1); err == nil {
		t.Fatalf("expected empty task token to fail")
	}
	if _, err := NewRuntimeClient("/tmp/worker.sock", "token", 0); err == nil {
		t.Fatalf("expected task id <= 0 to fail")
	}
}

func TestRuntimeClientRetriesUnavailable(t *testing.T) {
	stub := &workerRuntimeServiceStub{
		providerErrs: []error{
			status.Error(codes.Unavailable, "temporary unavailable"),
		},
	}
	socketPath := startWorkerRuntimeStubServer(t, stub)

	client, err := NewRuntimeClient(socketPath, "task-token-1", 7001)
	if err != nil {
		t.Fatalf("new runtime client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	resp, err := client.GetProviderConfig(context.Background(), 11, "subfinder")
	if err != nil {
		t.Fatalf("expected retry success, got err=%v", err)
	}
	if resp.Content == "" {
		t.Fatalf("expected provider content after retry")
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.providerCalls != 2 {
		t.Fatalf("expected 2 provider calls (1 retry), got %d", stub.providerCalls)
	}
}

func TestRuntimeClientDoesNotRetryAuthErrors(t *testing.T) {
	stub := &workerRuntimeServiceStub{
		providerErrs: []error{
			status.Error(codes.Unauthenticated, "invalid task token"),
		},
	}
	socketPath := startWorkerRuntimeStubServer(t, stub)

	client, err := NewRuntimeClient(socketPath, "task-token-1", 7001)
	if err != nil {
		t.Fatalf("new runtime client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	_, err = client.GetProviderConfig(context.Background(), 11, "subfinder")
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated error, got %v", err)
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.providerCalls != 1 {
		t.Fatalf("expected 1 provider call without retry, got %d", stub.providerCalls)
	}
}

func startWorkerRuntimeStubServer(t *testing.T, svc runtimev1.WorkerRuntimeServiceServer) string {
	t.Helper()

	socketPath := filepath.Join(os.TempDir(), "wrrtc-"+time.Now().Format("150405.000000000")+".sock")
	_ = os.Remove(socketPath)

	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix socket: %v", err)
	}
	server := grpc.NewServer()
	runtimev1.RegisterWorkerRuntimeServiceServer(server, svc)
	go func() {
		_ = server.Serve(lis)
	}()

	t.Cleanup(func() {
		server.GracefulStop()
		_ = lis.Close()
		_ = os.Remove(socketPath)
	})
	return socketPath
}
