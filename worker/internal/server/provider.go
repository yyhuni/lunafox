package server

import (
	"context"
	"fmt"
)

// GetProviderConfig fetches tool-specific configuration from Server
func (c *Client) GetProviderConfig(ctx context.Context, scanID int, toolName string) (*ProviderConfig, error) {
	url := fmt.Sprintf("%s/api/worker/scans/%d/provider-config?tool=%s", c.baseURL, scanID, toolName)
	return fetchJSON[*ProviderConfig](ctx, c, url)
}
