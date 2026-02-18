package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types/container"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Client wraps the Docker SDK client.
type Client struct {
	cli *client.Client
}

// NewClient creates a Docker client using environment configuration.
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli}, nil
}

// Close closes the Docker client.
func (c *Client) Close() error {
	return c.cli.Close()
}

// ImagePull pulls an image from the registry.
func (c *Client) ImagePull(ctx context.Context, imageRef string) (io.ReadCloser, error) {
	return c.cli.ImagePull(ctx, imageRef, imagetypes.PullOptions{})
}

// ContainerCreate creates a container.
func (c *Client) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, name string) (container.CreateResponse, error) {
	return c.cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, name)
}

// ContainerStart starts a container.
func (c *Client) ContainerStart(ctx context.Context, containerID string, opts container.StartOptions) error {
	return c.cli.ContainerStart(ctx, containerID, opts)
}
