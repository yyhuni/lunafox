package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"go.uber.org/zap"
)

type failingClient struct{}

func (f failingClient) GetProviderConfig(ctx context.Context, scanID int, toolName string) (*ProviderConfig, error) {
	return nil, nil
}

func (f failingClient) EnsureWordlistLocal(ctx context.Context, wordlistName, basePath string) (string, error) {
	return "", nil
}

func (f failingClient) PostBatch(ctx context.Context, scanID, targetID int, dataType string, items []any) error {
	return &HTTPError{StatusCode: 500, Body: "server error"}
}

type captureClient struct {
	calls     int
	failCount int
	err       error
	batches   [][]any
	onPost    func()
}

func (c *captureClient) GetProviderConfig(ctx context.Context, scanID int, toolName string) (*ProviderConfig, error) {
	return nil, nil
}

func (c *captureClient) EnsureWordlistLocal(ctx context.Context, wordlistName, basePath string) (string, error) {
	return "", nil
}

func (c *captureClient) PostBatch(ctx context.Context, scanID, targetID int, dataType string, items []any) error {
	c.calls++
	copied := append([]any(nil), items...)
	c.batches = append(c.batches, copied)
	if c.onPost != nil {
		c.onPost()
	}
	if c.calls <= c.failCount && c.err != nil {
		return c.err
	}
	return nil
}

func withNopLogger(t *testing.T) {
	t.Helper()
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() {
		pkg.Logger = prevLogger
	})
}

func TestBatchSender_RequeuesOnFailure(t *testing.T) {
	withNopLogger(t)
	ctx := context.Background()
	sender := NewBatchSender(ctx, failingClient{}, 1, 2, "subdomain", 2)

	require.NoError(t, sender.Add(map[string]string{"name": "a.example.com"}))
	err := sender.Add(map[string]string{"name": "b.example.com"})
	require.Error(t, err)

	sender.mu.Lock()
	defer sender.mu.Unlock()
	require.Len(t, sender.batch, 2)
}

func TestBatchSender_SendsOnBatchSize(t *testing.T) {
	withNopLogger(t)
	client := &captureClient{}
	sender := NewBatchSender(context.Background(), client, 1, 2, "subdomain", 2)

	require.NoError(t, sender.Add("a"))
	assert.Equal(t, 0, client.calls)

	require.NoError(t, sender.Add("b"))
	assert.Equal(t, 1, client.calls)
	require.Len(t, client.batches, 1)
	assert.Equal(t, []any{"a", "b"}, client.batches[0])

	items, batches := sender.Stats()
	assert.Equal(t, 2, items)
	assert.Equal(t, 1, batches)
}

func TestBatchSender_FlushSendsRemaining(t *testing.T) {
	withNopLogger(t)
	client := &captureClient{}
	sender := NewBatchSender(context.Background(), client, 1, 2, "subdomain", 3)

	require.NoError(t, sender.Add("a"))
	require.NoError(t, sender.Add("b"))
	require.NoError(t, sender.Flush())

	assert.Equal(t, 1, client.calls)
	require.Len(t, client.batches, 1)
	assert.Equal(t, []any{"a", "b"}, client.batches[0])

	items, batches := sender.Stats()
	assert.Equal(t, 2, items)
	assert.Equal(t, 1, batches)
}

func TestBatchSender_FlushEmpty(t *testing.T) {
	withNopLogger(t)
	client := &captureClient{}
	sender := NewBatchSender(context.Background(), client, 1, 2, "subdomain", 2)

	require.NoError(t, sender.Flush())
	assert.Equal(t, 0, client.calls)
}

func TestBatchSender_ContextCanceled(t *testing.T) {
	withNopLogger(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &captureClient{}
	sender := NewBatchSender(ctx, client, 1, 2, "subdomain", 2)

	err := sender.Add("a")
	require.ErrorIs(t, err, context.Canceled)
}

func TestBatchSender_RequeueThenSuccess(t *testing.T) {
	withNopLogger(t)
	client := &captureClient{
		failCount: 1,
		err:       &HTTPError{StatusCode: 500, Body: "server error"},
	}
	sender := NewBatchSender(context.Background(), client, 1, 2, "subdomain", 2)

	require.NoError(t, sender.Add("a"))
	err := sender.Add("b")
	require.Error(t, err)

	items, batches := sender.Stats()
	assert.Equal(t, 0, items)
	assert.Equal(t, 0, batches)

	require.NoError(t, sender.Flush())
	items, batches = sender.Stats()
	assert.Equal(t, 2, items)
	assert.Equal(t, 1, batches)
	assert.Equal(t, 2, client.calls)
}

func TestBatchSender_DefaultBatchSize(t *testing.T) {
	withNopLogger(t)
	sender := NewBatchSender(context.Background(), &captureClient{}, 1, 2, "subdomain", 0)
	assert.Equal(t, 1000, sender.batchSize)
	assert.Equal(t, 10000, sender.maxQueuedItems)
}

func TestBatchSender_QueueMetrics(t *testing.T) {
	withNopLogger(t)
	client := &captureClient{
		failCount: 1,
		err:       &HTTPError{StatusCode: 500, Body: "server error"},
	}
	sender := NewBatchSender(context.Background(), client, 1, 2, "subdomain", 2)

	require.NoError(t, sender.Add("a"))
	err := sender.Add("b")
	require.Error(t, err)

	queued, retries, dropped := sender.QueueMetrics()
	assert.Equal(t, 2, queued)
	assert.Equal(t, 1, retries)
	assert.Equal(t, 0, dropped)

	require.NoError(t, sender.Flush())
	queued, retries, dropped = sender.QueueMetrics()
	assert.Equal(t, 0, queued)
	assert.Equal(t, 1, retries)
	assert.Equal(t, 0, dropped)
}

func TestBatchSender_AddFailsFastWhenQueueLimitExceeded(t *testing.T) {
	withNopLogger(t)
	client := &captureClient{
		failCount: 1,
		err:       &HTTPError{StatusCode: 500, Body: "server error"},
	}
	sender := NewBatchSenderWithQueueLimit(context.Background(), client, 1, 2, "subdomain", 2, 2)

	require.NoError(t, sender.Add("a"))
	err := sender.Add("b")
	require.Error(t, err)

	err = sender.Add("c")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrBatchQueueOverflow)

	queued, retries, dropped := sender.QueueMetrics()
	assert.Equal(t, 2, queued)
	assert.Equal(t, 1, retries)
	assert.Equal(t, 1, dropped)
}

func TestBatchSender_RequeueOverflowDropsNewest(t *testing.T) {
	withNopLogger(t)
	client := &captureClient{
		failCount: 1,
		err:       &HTTPError{StatusCode: 500, Body: "server error"},
	}
	sender := NewBatchSenderWithQueueLimit(context.Background(), client, 1, 2, "subdomain", 2, 2)

	client.onPost = func() {
		client.onPost = nil
		require.NoError(t, sender.Add("c"))
	}

	require.NoError(t, sender.Add("a"))
	err := sender.Add("b")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrBatchQueueOverflow)

	sender.mu.Lock()
	require.Equal(t, []any{"a", "b"}, sender.batch)
	sender.mu.Unlock()

	queued, retries, dropped := sender.QueueMetrics()
	assert.Equal(t, 2, queued)
	assert.Equal(t, 1, retries)
	assert.Equal(t, 1, dropped)
}
