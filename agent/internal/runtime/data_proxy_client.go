package runtime

import (
	"context"
	"errors"
	"io"
	"strings"

	grpcauth "github.com/yyhuni/lunafox/agent/internal/grpc/runtime/auth"
	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	"google.golang.org/grpc"
)

func (c *Client) GetProviderConfig(ctx context.Context, scanID int, toolName string, taskID int) (string, error) {
	if scanID <= 0 {
		return "", errors.New("scan ID is required")
	}
	if strings.TrimSpace(toolName) == "" {
		return "", errors.New("tool name is required")
	}

	proxyClient, callCtx, err := c.dataProxyClient(ctx)
	if err != nil {
		return "", err
	}

	resp, err := proxyClient.GetProviderConfig(callCtx, &runtimev1.GetProviderConfigRequest{
		ScanId:   int32(scanID),
		ToolName: toolName,
		TaskId:   int32(taskID),
	})
	if err != nil {
		return "", err
	}
	return resp.GetContent(), nil
}

func (c *Client) GetWordlistMeta(ctx context.Context, wordlistName string, taskID int) (*WordlistMeta, error) {
	wordlistName = strings.TrimSpace(wordlistName)
	if wordlistName == "" {
		return nil, errors.New("wordlist name is required")
	}

	proxyClient, callCtx, err := c.dataProxyClient(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := proxyClient.GetWordlistMeta(callCtx, &runtimev1.GetWordlistMetaRequest{
		WordlistName: wordlistName,
		TaskId:       int32(taskID),
	})
	if err != nil {
		return nil, err
	}

	return &WordlistMeta{FileHash: resp.GetFileHash()}, nil
}

func (c *Client) DownloadWordlist(ctx context.Context, wordlistName string, taskID int, writer io.Writer) error {
	wordlistName = strings.TrimSpace(wordlistName)
	if wordlistName == "" {
		return errors.New("wordlist name is required")
	}
	if writer == nil {
		return errors.New("writer is required")
	}

	proxyClient, callCtx, err := c.dataProxyClient(ctx)
	if err != nil {
		return err
	}

	stream, err := proxyClient.DownloadWordlist(callCtx, &runtimev1.DownloadWordlistRequest{
		WordlistName: wordlistName,
		TaskId:       int32(taskID),
	})
	if err != nil {
		return err
	}

	for {
		chunk, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			return nil
		}
		if recvErr != nil {
			return recvErr
		}
		if chunk == nil || len(chunk.GetData()) == 0 {
			continue
		}
		if _, err := writer.Write(chunk.GetData()); err != nil {
			return err
		}
	}
}

func (c *Client) BatchUpsertAssets(
	ctx context.Context,
	scanID, targetID, taskID int,
	dataType string,
	itemsJSON []string,
) (int, error) {
	if scanID <= 0 || targetID <= 0 || taskID <= 0 {
		return 0, errors.New("scan ID, target ID, and task ID are required")
	}
	if strings.TrimSpace(dataType) == "" {
		return 0, errors.New("data type is required")
	}
	if len(itemsJSON) == 0 {
		return 0, errors.New("items json is required")
	}

	proxyClient, callCtx, err := c.dataProxyClient(ctx)
	if err != nil {
		return 0, err
	}

	resp, err := proxyClient.BatchUpsertAssets(callCtx, &runtimev1.BatchUpsertAssetsRequest{
		ScanId:    int32(scanID),
		TargetId:  int32(targetID),
		TaskId:    int32(taskID),
		DataType:  dataType,
		ItemsJson: itemsJSON,
	})
	if err != nil {
		return 0, err
	}
	return int(resp.GetAccepted()), nil
}

func (c *Client) dataProxyClient(ctx context.Context) (runtimev1.AgentDataProxyServiceClient, context.Context, error) {
	conn := c.currentConn()
	if conn == nil {
		return nil, nil, ErrRuntimeNotConnected
	}
	callCtx := grpcauth.WithAgentKey(ctx, c.apiKey)
	return runtimev1.NewAgentDataProxyServiceClient(conn), callCtx, nil
}

func (c *Client) currentConn() *grpc.ClientConn {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.conn
}
