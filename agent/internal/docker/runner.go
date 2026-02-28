package docker

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/yyhuni/lunafox/agent/internal/domain"
)

const (
	sharedDataVolumeBindEnvKey = "LUNAFOX_SHARED_DATA_VOLUME_BIND"
	defaultSharedDataMountPath = "/opt/lunafox"
	runtimeVolumeNameEnvKey    = "LUNAFOX_RUNTIME_VOLUME"
	defaultRuntimeVolumeName   = "lunafox_runtime"
	defaultRuntimeMountPath    = "/run/lunafox"
)

// StartWorker starts a worker container for a task and returns the container ID.
func (c *Client) StartWorker(ctx context.Context, t *domain.Task, agentSocket, taskToken string) (string, error) {
	if t == nil {
		return "", fmt.Errorf("task is nil")
	}
	if strings.TrimSpace(agentSocket) == "" {
		return "", fmt.Errorf("agent socket is required")
	}
	if strings.TrimSpace(taskToken) == "" {
		return "", fmt.Errorf("task token is required")
	}
	if err := os.MkdirAll(t.WorkspaceDir, 0755); err != nil {
		return "", fmt.Errorf("prepare workspace: %w", err)
	}

	image, err := resolveWorkerImage()
	if err != nil {
		return "", err
	}
	sharedDataVolumeBind, err := resolveSharedDataVolumeBind()
	if err != nil {
		return "", err
	}
	runtimeVolumeBind, err := resolveRuntimeVolumeBind()
	if err != nil {
		return "", err
	}
	env := buildWorkerEnv(t, agentSocket, taskToken)

	config := &container.Config{
		Image: image,
		Env:   env,
		Cmd:   strslice.StrSlice{},
	}

	hostConfig := &container.HostConfig{
		Binds:       []string{sharedDataVolumeBind, runtimeVolumeBind},
		AutoRemove:  false,
		OomScoreAdj: 500,
	}

	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, &network.NetworkingConfig{}, nil, "")
	if err != nil {
		return "", err
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", err
	}

	return resp.ID, nil
}

func resolveWorkerImage() (string, error) {
	imageRef := strings.TrimSpace(os.Getenv("WORKER_IMAGE_REF"))
	if imageRef == "" {
		return "", fmt.Errorf("WORKER_IMAGE_REF environment variable is required")
	}
	if !hasImageTagOrDigest(imageRef) {
		return "", fmt.Errorf("WORKER_IMAGE_REF must include tag or digest")
	}
	return imageRef, nil
}

func hasImageTagOrDigest(imageRef string) bool {
	if strings.Contains(imageRef, "@") {
		return true
	}
	return strings.LastIndex(imageRef, ":") > strings.LastIndex(imageRef, "/")
}

func resolveSharedDataVolumeBind() (string, error) {
	raw := strings.TrimSpace(os.Getenv(sharedDataVolumeBindEnvKey))
	if raw == "" {
		return "", fmt.Errorf("%s environment variable is required", sharedDataVolumeBindEnvKey)
	}

	parts := strings.Split(raw, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return "", fmt.Errorf("%s must be '<named-volume>:%s[:mode]'", sharedDataVolumeBindEnvKey, defaultSharedDataMountPath)
	}

	source := strings.TrimSpace(parts[0])
	target := strings.TrimSpace(parts[1])
	if source == "" {
		return "", fmt.Errorf("%s source is empty", sharedDataVolumeBindEnvKey)
	}
	if !isValidNamedVolumeName(source) {
		return "", fmt.Errorf("%s source must be a Docker named volume", sharedDataVolumeBindEnvKey)
	}
	if target != defaultSharedDataMountPath {
		return "", fmt.Errorf("%s target must be %s", sharedDataVolumeBindEnvKey, defaultSharedDataMountPath)
	}

	if len(parts) == 3 {
		mode := strings.TrimSpace(parts[2])
		if mode == "" {
			return "", fmt.Errorf("%s mode is empty", sharedDataVolumeBindEnvKey)
		}
	}
	return raw, nil
}

func resolveRuntimeVolumeBind() (string, error) {
	volumeName := defaultRuntimeVolumeName
	if !isValidNamedVolumeName(volumeName) {
		return "", fmt.Errorf("%s must be a Docker named volume", runtimeVolumeNameEnvKey)
	}
	return fmt.Sprintf("%s:%s:ro", volumeName, defaultRuntimeMountPath), nil
}

func isValidNamedVolumeName(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	for idx, r := range trimmed {
		if idx == 0 {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return false
			}
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '_', '.', '-':
			continue
		default:
			return false
		}
	}
	return true
}

func buildWorkerEnv(t *domain.Task, agentSocket, taskToken string) []string {
	return []string{
		fmt.Sprintf("TASK_ID=%d", t.ID),
		fmt.Sprintf("SCAN_ID=%d", t.ScanID),
		fmt.Sprintf("TARGET_ID=%d", t.TargetID),
		fmt.Sprintf("TARGET_NAME=%s", t.TargetName),
		fmt.Sprintf("TARGET_TYPE=%s", t.TargetType),
		fmt.Sprintf("WORKFLOW_NAME=%s", t.WorkflowName),
		fmt.Sprintf("WORKSPACE_DIR=%s", t.WorkspaceDir),
		fmt.Sprintf("CONFIG=%s", t.Config),
		fmt.Sprintf("AGENT_SOCKET=%s", agentSocket),
		fmt.Sprintf("TASK_TOKEN=%s", taskToken),
	}
}
