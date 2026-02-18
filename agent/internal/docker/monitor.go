package docker

import (
	"context"

	"github.com/docker/docker/api/types/container"
)

// Wait waits for a container to stop and returns the exit code.
func (c *Client) Wait(ctx context.Context, containerID string) (int64, error) {
	statusCh, errCh := c.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case status := <-statusCh:
		return status.StatusCode, nil
	case err := <-errCh:
		return 0, err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}
