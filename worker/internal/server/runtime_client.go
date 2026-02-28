package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	grpcauth "github.com/yyhuni/lunafox/worker/internal/grpc/runtime/auth"
	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	defaultRuntimeDialTimeout = 5 * time.Second
	defaultRetryBaseDelay     = 10 * time.Millisecond
	defaultRetryAttempts      = 3
)

type RuntimeClient struct {
	conn      *grpc.ClientConn
	client    runtimev1.WorkerRuntimeServiceClient
	taskToken string
	taskID    int
}

func NewRuntimeClient(socketPath, taskToken string, taskID int) (*RuntimeClient, error) {
	socketPath = strings.TrimSpace(socketPath)
	taskToken = strings.TrimSpace(taskToken)
	if socketPath == "" {
		return nil, errors.New("agent socket is required")
	}
	if taskToken == "" {
		return nil, errors.New("task token is required")
	}
	if taskID <= 0 {
		return nil, errors.New("task id must be greater than 0")
	}

	dialCtx, cancel := context.WithTimeout(context.Background(), defaultRuntimeDialTimeout)
	defer cancel()
	conn, err := grpc.DialContext(
		dialCtx,
		"unix://"+socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("dial worker runtime socket: %w", err)
	}

	return &RuntimeClient{
		conn:      conn,
		client:    runtimev1.NewWorkerRuntimeServiceClient(conn),
		taskToken: taskToken,
		taskID:    taskID,
	}, nil
}

func (c *RuntimeClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *RuntimeClient) GetProviderConfig(ctx context.Context, scanID int, toolName string) (*ProviderConfig, error) {
	if scanID <= 0 {
		return nil, errors.New("scan id must be greater than 0")
	}
	toolName = strings.TrimSpace(toolName)
	if toolName == "" {
		return nil, errors.New("tool name is required")
	}

	req := &runtimev1.GetProviderConfigRequest{
		ScanId:   int32(scanID),
		ToolName: toolName,
		TaskId:   int32(c.taskID),
	}
	var resp *runtimev1.GetProviderConfigResponse
	if err := c.withRetry(ctx, func(callCtx context.Context) error {
		result, err := c.client.GetProviderConfig(c.withTaskToken(callCtx), req)
		if err != nil {
			return err
		}
		resp = result
		return nil
	}); err != nil {
		return nil, err
	}
	return &ProviderConfig{Content: resp.GetContent()}, nil
}

func (c *RuntimeClient) EnsureWordlistLocal(ctx context.Context, wordlistName, basePath string) (string, error) {
	wordlistName = strings.TrimSpace(wordlistName)
	basePath = strings.TrimSpace(basePath)
	if wordlistName == "" {
		return "", errors.New("wordlist name is required")
	}
	if basePath == "" {
		return "", errors.New("base path is required")
	}

	req := &runtimev1.EnsureWordlistRequest{
		TaskId:        int32(c.taskID),
		WordlistName:  wordlistName,
		LocalBasePath: basePath,
	}
	var resp *runtimev1.EnsureWordlistResponse
	if err := c.withRetry(ctx, func(callCtx context.Context) error {
		result, err := c.client.EnsureWordlist(c.withTaskToken(callCtx), req)
		if err != nil {
			return err
		}
		resp = result
		return nil
	}); err != nil {
		return "", err
	}
	return resp.GetLocalPath(), nil
}

func (c *RuntimeClient) PostBatch(ctx context.Context, scanID, targetID int, dataType string, items []any) error {
	if scanID <= 0 || targetID <= 0 {
		return errors.New("scan id and target id must be greater than 0")
	}
	dataType = strings.TrimSpace(dataType)
	if dataType == "" {
		return errors.New("data type is required")
	}
	if len(items) == 0 {
		return errors.New("items must not be empty")
	}

	itemsJSON, err := marshalItemsJSON(items)
	if err != nil {
		return err
	}

	req := &runtimev1.PostBatchRequest{
		ScanId:    int32(scanID),
		TargetId:  int32(targetID),
		TaskId:    int32(c.taskID),
		DataType:  dataType,
		ItemsJson: itemsJSON,
	}

	return c.withRetry(ctx, func(callCtx context.Context) error {
		_, err := c.client.PostBatch(c.withTaskToken(callCtx), req)
		return err
	})
}

func (c *RuntimeClient) withTaskToken(ctx context.Context) context.Context {
	return grpcauth.WithTaskToken(ctx, c.taskToken)
}

func marshalItemsJSON(items []any) ([]string, error) {
	result := make([]string, 0, len(items))
	for _, item := range items {
		payload, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("marshal batch item: %w", err)
		}
		result = append(result, string(payload))
	}
	return result, nil
}

func (c *RuntimeClient) withRetry(ctx context.Context, call func(context.Context) error) error {
	var lastErr error
	for attempt := 1; attempt <= defaultRetryAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := call(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
		if !isRetryableGRPCError(err) || attempt == defaultRetryAttempts {
			return err
		}

		backoff := defaultRetryBaseDelay * time.Duration(1<<(attempt-1))
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}

func isRetryableGRPCError(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return true
	}
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Aborted, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}
