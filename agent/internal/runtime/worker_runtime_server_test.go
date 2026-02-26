package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	grpcauth "github.com/yyhuni/lunafox/agent/internal/grpc/runtime/auth"
	runtimev1 "github.com/yyhuni/lunafox/agent/internal/grpc/runtime/v1/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type workerRuntimeProxyStub struct {
	getProviderConfigContent string
	getProviderConfigErr     error
	getProviderConfigTaskID  int

	wordlistMeta       *WordlistMeta
	getWordlistMetaErr error
	getWordlistTaskID  int

	downloadBytes       []byte
	downloadErr         error
	downloadCallCount   int
	downloadWordlistArg string
	downloadTaskIDArg   int

	postBatchAccepted int
	postBatchErr      error
	postBatchTaskID   int
	postBatchDataType string
	postBatchItems    []string
}

func (stub *workerRuntimeProxyStub) GetProviderConfig(_ context.Context, _ int, _ string, taskID int) (string, error) {
	stub.getProviderConfigTaskID = taskID
	if stub.getProviderConfigErr != nil {
		return "", stub.getProviderConfigErr
	}
	return stub.getProviderConfigContent, nil
}

func (stub *workerRuntimeProxyStub) GetWordlistMeta(_ context.Context, _ string, taskID int) (*WordlistMeta, error) {
	stub.getWordlistTaskID = taskID
	if stub.getWordlistMetaErr != nil {
		return nil, stub.getWordlistMetaErr
	}
	if stub.wordlistMeta == nil {
		return nil, errors.New("wordlist meta missing")
	}
	return stub.wordlistMeta, nil
}

func (stub *workerRuntimeProxyStub) DownloadWordlist(_ context.Context, wordlistName string, taskID int, writer io.Writer) error {
	stub.downloadCallCount++
	stub.downloadWordlistArg = wordlistName
	stub.downloadTaskIDArg = taskID
	if stub.downloadErr != nil {
		return stub.downloadErr
	}
	if len(stub.downloadBytes) == 0 {
		return nil
	}
	_, err := writer.Write(stub.downloadBytes)
	return err
}

func (stub *workerRuntimeProxyStub) BatchUpsertAssets(_ context.Context, _, _, taskID int, dataType string, itemsJSON []string) (int, error) {
	stub.postBatchTaskID = taskID
	stub.postBatchDataType = dataType
	stub.postBatchItems = append([]string(nil), itemsJSON...)
	if stub.postBatchErr != nil {
		return 0, stub.postBatchErr
	}
	return stub.postBatchAccepted, nil
}

func TestWorkerRuntimeUDSServerIntegration(t *testing.T) {
	wordlistContent := []byte("admin\napi\n")
	wordlistHash := sha256Hex(wordlistContent)
	proxy := &workerRuntimeProxyStub{
		getProviderConfigContent: "sources:\n- crtsh",
		wordlistMeta:             &WordlistMeta{FileHash: wordlistHash},
		downloadBytes:            wordlistContent,
		postBatchAccepted:        2,
	}

	client := startWorkerRuntimeTestServer(t, proxy)
	authCtx := metadata.AppendToOutgoingContext(context.Background(), grpcauth.TaskTokenHeader, "task-token-1")

	cfgResp, err := client.GetProviderConfig(authCtx, &runtimev1.GetProviderConfigRequest{
		ScanId:   11,
		ToolName: "subfinder",
		TaskId:   7001,
	})
	if err != nil {
		t.Fatalf("get provider config failed: %v", err)
	}
	if cfgResp.GetContent() != "sources:\n- crtsh" {
		t.Fatalf("unexpected provider config content: %q", cfgResp.GetContent())
	}
	if proxy.getProviderConfigTaskID != 7001 {
		t.Fatalf("unexpected task id forwarded for provider config: %d", proxy.getProviderConfigTaskID)
	}

	basePath := filepath.Join(t.TempDir(), "wordlists")
	ensureResp, err := client.EnsureWordlist(authCtx, &runtimev1.EnsureWordlistRequest{
		TaskId:        7001,
		WordlistName:  "subs.txt",
		LocalBasePath: basePath,
	})
	if err != nil {
		t.Fatalf("ensure wordlist failed: %v", err)
	}
	expectedPath := filepath.Join(basePath, "subs.txt")
	if ensureResp.GetLocalPath() != expectedPath {
		t.Fatalf("unexpected local path: %q", ensureResp.GetLocalPath())
	}
	if ensureResp.GetFileHash() != wordlistHash {
		t.Fatalf("unexpected file hash: %q", ensureResp.GetFileHash())
	}
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read ensured wordlist: %v", err)
	}
	if string(data) != string(wordlistContent) {
		t.Fatalf("unexpected local wordlist content: %q", string(data))
	}

	postResp, err := client.PostBatch(authCtx, &runtimev1.PostBatchRequest{
		ScanId:    11,
		TargetId:  22,
		TaskId:    7001,
		DataType:  "subdomain",
		ItemsJson: []string{`{"name":"api.example.com"}`, `{"name":"www.example.com"}`},
	})
	if err != nil {
		t.Fatalf("post batch failed: %v", err)
	}
	if postResp.GetSent() != 2 {
		t.Fatalf("unexpected sent count: %d", postResp.GetSent())
	}
	if proxy.postBatchTaskID != 7001 || proxy.postBatchDataType != "subdomain" {
		t.Fatalf("unexpected post batch forwarding data: task=%d type=%q", proxy.postBatchTaskID, proxy.postBatchDataType)
	}
	if len(proxy.postBatchItems) != 2 {
		t.Fatalf("unexpected post batch item count: %d", len(proxy.postBatchItems))
	}

	_, err = client.GetProviderConfig(context.Background(), &runtimev1.GetProviderConfigRequest{
		ScanId:   11,
		ToolName: "subfinder",
		TaskId:   7001,
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated when task token missing, got=%v", err)
	}
}

func TestWorkerRuntimeEnsureWordlistHashMismatch(t *testing.T) {
	proxy := &workerRuntimeProxyStub{
		wordlistMeta:      &WordlistMeta{FileHash: sha256Hex([]byte("expected"))},
		downloadBytes:     []byte("actual"),
		postBatchAccepted: 1,
	}
	client := startWorkerRuntimeTestServer(t, proxy)
	authCtx := metadata.AppendToOutgoingContext(context.Background(), grpcauth.TaskTokenHeader, "task-token-2")

	_, err := client.EnsureWordlist(authCtx, &runtimev1.EnsureWordlistRequest{
		TaskId:        8002,
		WordlistName:  "subs.txt",
		LocalBasePath: t.TempDir(),
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal on hash mismatch, got=%v", err)
	}
}

func startWorkerRuntimeTestServer(t *testing.T, proxy WorkerRuntimeProxy) runtimev1.WorkerRuntimeServiceClient {
	t.Helper()

	socketPath := filepath.Join(os.TempDir(), "lfwrs-"+time.Now().Format("150405.000000000")+".sock")
	_ = os.Remove(socketPath)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunWorkerRuntimeServer(ctx, socketPath, proxy)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		select {
		case serveErr := <-errCh:
			cancel()
			t.Fatalf("worker runtime server failed before socket ready: %v", serveErr)
		default:
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("timeout waiting for socket file %s", socketPath)
		}
		time.Sleep(10 * time.Millisecond)
	}
	socketInfo, err := os.Stat(socketPath)
	if err != nil {
		cancel()
		t.Fatalf("stat socket: %v", err)
	}
	if socketInfo.Mode().Perm() != 0o666 {
		cancel()
		t.Fatalf("expected socket permissions 0666, got %#o", socketInfo.Mode().Perm())
	}

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(dialCancel)
	conn, err := grpc.DialContext(
		dialCtx,
		"unix://"+socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		cancel()
		t.Fatalf("dial worker runtime uds failed: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
		cancel()
		_ = os.Remove(socketPath)
		select {
		case serveErr := <-errCh:
			if serveErr != nil {
				t.Fatalf("worker runtime server exited with error: %v", serveErr)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting worker runtime server shutdown")
		}
	})
	return runtimev1.NewWorkerRuntimeServiceClient(conn)
}

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
