package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	grpcauth "github.com/yyhuni/lunafox/agent/internal/grpc/runtime/auth"
	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	"github.com/yyhuni/lunafox/contracts/runtimecontract"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const DefaultWorkerRuntimeSocketPath = runtimecontract.DefaultRuntimeMountPath + "/" + runtimecontract.WorkerRuntimeSocketName

type WordlistMeta struct {
	FileHash string
}

type WorkerRuntimeProxy interface {
	GetProviderConfig(ctx context.Context, scanID int, toolName string, taskID int) (string, error)
	GetWordlistMeta(ctx context.Context, wordlistName string, taskID int) (*WordlistMeta, error)
	DownloadWordlist(ctx context.Context, wordlistName string, taskID int, writer io.Writer) error
	BatchUpsertAssets(ctx context.Context, scanID, targetID, taskID int, dataType string, itemsJSON []string) (int, error)
}

type TaskSessionValidator interface {
	ValidateTaskSession(taskID int, taskToken string) bool
}

type workerRuntimeService struct {
	runtimev1.UnimplementedWorkerRuntimeServiceServer
	proxy            WorkerRuntimeProxy
	sessionValidator TaskSessionValidator
}

func RunWorkerRuntimeServer(ctx context.Context, socketPath string, proxy WorkerRuntimeProxy) error {
	if proxy == nil {
		return errors.New("worker runtime proxy is required")
	}
	sessionValidator, ok := proxy.(TaskSessionValidator)
	if !ok || sessionValidator == nil {
		return errors.New("worker runtime proxy must implement task session validator")
	}

	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		socketPath = DefaultWorkerRuntimeSocketPath
	}

	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		return fmt.Errorf("prepare runtime socket directory: %w", err)
	}

	if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale runtime socket: %w", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen worker runtime socket: %w", err)
	}
	if err := os.Chmod(socketPath, 0666); err != nil {
		_ = listener.Close()
		_ = os.Remove(socketPath)
		return fmt.Errorf("set runtime socket permissions: %w", err)
	}
	defer func() {
		_ = listener.Close()
		_ = os.Remove(socketPath)
	}()

	server := grpc.NewServer()
	runtimev1.RegisterWorkerRuntimeServiceServer(server, &workerRuntimeService{
		proxy:            proxy,
		sessionValidator: sessionValidator,
	})

	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()

	err = server.Serve(listener)
	if err == nil || errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}
	if ctx.Err() != nil && errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (s *workerRuntimeService) GetProviderConfig(ctx context.Context, req *runtimev1.GetProviderConfigRequest) (*runtimev1.GetProviderConfigResponse, error) {
	taskToken, err := requireTaskToken(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.ScanId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "scan_id is required")
	}
	if strings.TrimSpace(req.ToolName) == "" {
		return nil, status.Error(codes.InvalidArgument, "tool_name is required")
	}
	if req.TaskId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "task_id is required")
	}
	if err := s.requireTaskScope(int(req.TaskId), taskToken); err != nil {
		return nil, err
	}

	content, err := s.proxy.GetProviderConfig(ctx, int(req.ScanId), req.ToolName, int(req.TaskId))
	if err != nil {
		return nil, mapProxyError(err)
	}
	return &runtimev1.GetProviderConfigResponse{Content: content}, nil
}

func (s *workerRuntimeService) EnsureWordlist(ctx context.Context, req *runtimev1.EnsureWordlistRequest) (*runtimev1.EnsureWordlistResponse, error) {
	taskToken, err := requireTaskToken(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.TaskId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "task_id is required")
	}
	wordlistName := strings.TrimSpace(req.WordlistName)
	if wordlistName == "" {
		return nil, status.Error(codes.InvalidArgument, "wordlist_name is required")
	}
	basePath := strings.TrimSpace(req.LocalBasePath)
	if basePath == "" {
		return nil, status.Error(codes.InvalidArgument, "local_base_path is required")
	}
	if err := s.requireTaskScope(int(req.TaskId), taskToken); err != nil {
		return nil, err
	}

	meta, err := s.proxy.GetWordlistMeta(ctx, wordlistName, int(req.TaskId))
	if err != nil {
		return nil, mapProxyError(err)
	}
	expectedHash := strings.TrimSpace(meta.FileHash)
	if expectedHash == "" {
		return nil, status.Error(codes.Internal, "wordlist meta missing file hash")
	}

	localPath := filepath.Join(basePath, wordlistName)
	if actualHash, ok := verifyLocalFileHash(localPath, expectedHash); ok {
		return &runtimev1.EnsureWordlistResponse{
			LocalPath: localPath,
			FileHash:  actualHash,
		}, nil
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	tempPath := localPath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	downloadErr := s.proxy.DownloadWordlist(ctx, wordlistName, int(req.TaskId), file)
	closeErr := file.Close()
	if downloadErr != nil {
		_ = os.Remove(tempPath)
		return nil, mapProxyError(downloadErr)
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return nil, status.Error(codes.Internal, closeErr.Error())
	}

	if err := os.Rename(tempPath, localPath); err != nil {
		_ = os.Remove(tempPath)
		return nil, status.Error(codes.Internal, err.Error())
	}

	actualHash, err := calcFileHash(localPath)
	if err != nil {
		_ = os.Remove(localPath)
		return nil, status.Error(codes.Internal, err.Error())
	}
	if actualHash != expectedHash {
		_ = os.Remove(localPath)
		return nil, status.Error(codes.Internal, "downloaded wordlist hash mismatch")
	}

	return &runtimev1.EnsureWordlistResponse{
		LocalPath: localPath,
		FileHash:  actualHash,
	}, nil
}

func (s *workerRuntimeService) PostBatch(ctx context.Context, req *runtimev1.PostBatchRequest) (*runtimev1.PostBatchResponse, error) {
	taskToken, err := requireTaskToken(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.ScanId <= 0 || req.TargetId <= 0 || req.TaskId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "scan_id, target_id, and task_id are required")
	}
	if strings.TrimSpace(req.DataType) == "" {
		return nil, status.Error(codes.InvalidArgument, "data_type is required")
	}
	if len(req.ItemsJson) == 0 {
		return nil, status.Error(codes.InvalidArgument, "items_json must not be empty")
	}
	if err := s.requireTaskScope(int(req.TaskId), taskToken); err != nil {
		return nil, err
	}

	accepted, err := s.proxy.BatchUpsertAssets(
		ctx,
		int(req.ScanId),
		int(req.TargetId),
		int(req.TaskId),
		req.DataType,
		req.ItemsJson,
	)
	if err != nil {
		return nil, mapProxyError(err)
	}
	return &runtimev1.PostBatchResponse{Sent: int32(accepted)}, nil
}

func requireTaskToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "invalid task token")
	}

	values := md.Get(grpcauth.TaskTokenHeader)
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "invalid task token")
	}
	taskToken := strings.TrimSpace(values[0])
	if taskToken == "" {
		return "", status.Error(codes.Unauthenticated, "invalid task token")
	}
	return taskToken, nil
}

func (s *workerRuntimeService) requireTaskScope(taskID int, taskToken string) error {
	if taskID <= 0 {
		return status.Error(codes.InvalidArgument, "task_id is required")
	}
	if s.sessionValidator == nil || !s.sessionValidator.ValidateTaskSession(taskID, taskToken) {
		return status.Error(codes.PermissionDenied, "task scope mismatch")
	}
	return nil
}

func verifyLocalFileHash(path, expectedHash string) (string, bool) {
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	actualHash, err := calcFileHash(path)
	if err != nil {
		return "", false
	}
	return actualHash, actualHash == expectedHash
}

func calcFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func mapProxyError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrRuntimeNotConnected) {
		return status.Error(codes.Unavailable, err.Error())
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	return status.Error(codes.Internal, err.Error())
}
