package docker

import (
	"context"

	"github.com/docker/docker/api/types/container"
)

// Remove removes the container.
func (c *Client) Remove(ctx context.Context, containerID string) error {
	return c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
}

// Stop stops a running container with a timeout.
func (c *Client) Stop(ctx context.Context, containerID string) error {
	timeout := 10
	return c.cli.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	})
}
