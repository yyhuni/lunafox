package results

import (
	"context"
	"testing"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"github.com/yyhuni/lunafox/worker/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type captureClient struct {
	t       *testing.T
	batches [][]Subdomain
}

func (c *captureClient) GetProviderConfig(ctx context.Context, scanID int, toolName string) (*server.ProviderConfig, error) {
	return nil, nil
}

func (c *captureClient) EnsureWordlistLocal(ctx context.Context, wordlistName, basePath string) (string, error) {
	return "", nil
}

func (c *captureClient) PostBatch(ctx context.Context, scanID, targetID int, dataType string, items []any) error {
	require.Equal(c.t, "subdomain", dataType)
	batch := make([]Subdomain, 0, len(items))
	for _, item := range items {
		sd, ok := item.(Subdomain)
		require.True(c.t, ok)
		batch = append(batch, sd)
	}
	c.batches = append(c.batches, batch)
	return nil
}

func TestWriteSubdomains(t *testing.T) {
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() {
		pkg.Logger = prevLogger
	})

	ch := make(chan Subdomain, 2)
	ch <- Subdomain{Name: "a.example.com"}
	ch <- Subdomain{Name: "b.example.com"}
	close(ch)

	client := &captureClient{t: t}
	items, batches, err := WriteSubdomains(context.Background(), client, 1, 2, ch)
	require.NoError(t, err)
	require.Equal(t, 2, items)
	require.Equal(t, 1, batches)
	require.Len(t, client.batches, 1)
	require.Equal(t, []Subdomain{{Name: "a.example.com"}, {Name: "b.example.com"}}, client.batches[0])
}

type failingBatchClient struct {
	err error
}

func (c failingBatchClient) GetProviderConfig(ctx context.Context, scanID int, toolName string) (*server.ProviderConfig, error) {
	return nil, nil
}

func (c failingBatchClient) EnsureWordlistLocal(ctx context.Context, wordlistName, basePath string) (string, error) {
	return "", nil
}

func (c failingBatchClient) PostBatch(ctx context.Context, scanID, targetID int, dataType string, items []any) error {
	return c.err
}

func TestWriteSubdomains_EmptyInput(t *testing.T) {
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() {
		pkg.Logger = prevLogger
	})

	ch := make(chan Subdomain)
	close(ch)

	items, batches, err := WriteSubdomains(context.Background(), &captureClient{t: t}, 1, 2, ch)
	require.NoError(t, err)
	assert.Equal(t, 0, items)
	assert.Equal(t, 0, batches)
}

func TestWriteSubdomains_PropagatesBatchError(t *testing.T) {
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() {
		pkg.Logger = prevLogger
	})

	ch := make(chan Subdomain, 1)
	ch <- Subdomain{Name: "a.example.com"}
	close(ch)

	err := &server.HTTPError{StatusCode: 500, Body: "server error"}
	items, batches, got := WriteSubdomains(context.Background(), failingBatchClient{err: err}, 1, 2, ch)
	require.Error(t, got)
	assert.Equal(t, 0, items)
	assert.Equal(t, 0, batches)
}

func TestWriteSubdomains_ContextCanceled(t *testing.T) {
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() {
		pkg.Logger = prevLogger
	})

	ch := make(chan Subdomain, 1)
	ch <- Subdomain{Name: "a.example.com"}
	close(ch)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	items, batches, err := WriteSubdomains(ctx, &captureClient{t: t}, 1, 2, ch)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, items)
	assert.Equal(t, 0, batches)
}
