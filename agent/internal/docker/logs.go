package docker

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
)

const (
	maxErrorBytes = 4096
)

// TailLogs returns the last N lines of container logs, truncated to 4KB.
func (c *Client) TailLogs(ctx context.Context, containerID string, lines int) (string, error) {
	reader, err := c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: false,
		Tail:       strconv.Itoa(lines),
	})
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return "", err
	}

	out := buf.String()
	out = strings.TrimSpace(out)
	if len(out) > maxErrorBytes {
		out = out[len(out)-maxErrorBytes:]
	}
	return out, nil
}

// TruncateErrorMessage clamps message length to 4KB.
func TruncateErrorMessage(message string) string {
	if len(message) <= maxErrorBytes {
		return message
	}
	return message[:maxErrorBytes]
}
