package server

import (
	"context"
	"net/http"
	"testing"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetProviderConfig(t *testing.T) {
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() { pkg.Logger = prevLogger })

	client := &Client{
		baseURL: "http://example",
		token:   "token",
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "/api/worker/scans/9/provider-config", r.URL.Path)
				assert.Equal(t, "subfinder", r.URL.Query().Get("tool"))
				return newResponse(http.StatusOK, `{"content":"config"}`), nil
			}),
		},
	}

	cfg, err := client.GetProviderConfig(context.Background(), 9, "subfinder")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "config", cfg.Content)
}
