package infrastructure

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/application"
)

const (
	requestSchema      = 1
	defaultUpgradeRoot = "/opt/lunafox"
	defaultSocketName  = ".lunafox/upgrade/upgrader.sock"
)

// HostUpgradeDispatcher is the narrow Server-side client for the independent
// host upgrader. The wire payload intentionally has no command, image or path
// fields; those values are owned by the host process and its fixed manifest.
type HostUpgradeDispatcher struct {
	socketPath string
	timeout    time.Duration
}

type wireRequest struct {
	SchemaVersion  int    `json:"schemaVersion"`
	OperationID    string `json:"operationId"`
	Action         string `json:"action"`
	ManifestDigest string `json:"manifestDigest"`
}

type wireResponse struct {
	Accepted bool   `json:"accepted"`
	Error    string `json:"error,omitempty"`
}

func NewHostUpgradeDispatcher(socketPath string) (*HostUpgradeDispatcher, error) {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return nil, fmt.Errorf("upgrade socket path is required")
	}
	return &HostUpgradeDispatcher{socketPath: filepath.Clean(socketPath), timeout: 15 * time.Second}, nil
}

func NewDefaultHostUpgradeDispatcher() (*HostUpgradeDispatcher, error) {
	root := strings.TrimSpace(os.Getenv("LUNAFOX_UPGRADE_DEPLOYMENT_ROOT"))
	if root == "" {
		root = defaultUpgradeRoot
	}
	return NewHostUpgradeDispatcher(filepath.Join(root, defaultSocketName))
}

func (dispatcher *HostUpgradeDispatcher) Dispatch(ctx context.Context, request application.HostUpgradeRequest) error {
	if dispatcher == nil || dispatcher.socketPath == "" {
		return fmt.Errorf("host upgrade dispatcher is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if request.OperationID == "" || request.ManifestDigest == "" {
		return fmt.Errorf("host upgrade request identity is required")
	}
	if request.Action != application.HostUpgradeActionStart && request.Action != application.HostUpgradeActionResume && request.Action != application.HostUpgradeActionRepair {
		return fmt.Errorf("unsupported host upgrade action %q", request.Action)
	}
	deadline := dispatcher.timeout
	if deadline <= 0 {
		deadline = 15 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	connection, err := (&net.Dialer{}).DialContext(callCtx, "unix", dispatcher.socketPath)
	if err != nil {
		return fmt.Errorf("connect to host upgrader: %w", err)
	}
	defer func() { _ = connection.Close() }()
	_ = connection.SetDeadline(time.Now().Add(deadline))
	payload, err := json.Marshal(wireRequest{
		SchemaVersion:  requestSchema,
		OperationID:    request.OperationID,
		Action:         string(request.Action),
		ManifestDigest: request.ManifestDigest,
	})
	if err != nil {
		return fmt.Errorf("encode host upgrade request: %w", err)
	}
	if _, err := connection.Write(append(payload, '\n')); err != nil {
		return fmt.Errorf("send host upgrade request: %w", err)
	}
	var response wireResponse
	if err := json.NewDecoder(bufio.NewReader(connection)).Decode(&response); err != nil {
		return fmt.Errorf("decode host upgrade response: %w", err)
	}
	if !response.Accepted {
		message := strings.TrimSpace(response.Error)
		if message == "" {
			message = "host upgrader rejected the request"
		}
		return fmt.Errorf("%s", message)
	}
	return nil
}

var _ application.HostUpgradeDispatcher = (*HostUpgradeDispatcher)(nil)
