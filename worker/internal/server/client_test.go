package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{},
	}
}

func init() {
	if pkg.Logger == nil {
		pkg.Logger = zap.NewNop()
	}
}

func TestHTTPErrorIsRetryable(t *testing.T) {
	assert.True(t, (&HTTPError{StatusCode: 500}).IsRetryable())
	assert.False(t, (&HTTPError{StatusCode: 400}).IsRetryable())
}

func TestIsRetryableError(t *testing.T) {
	assert.False(t, isRetryableError(nil))
	assert.False(t, isRetryableError(&HTTPError{StatusCode: 400}))
	assert.True(t, isRetryableError(&HTTPError{StatusCode: 503}))
	assert.True(t, isRetryableError(errors.New("network error")))
}

func TestDoRequestSuccess(t *testing.T) {
	client := &Client{
		token: "token",
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "token", r.Header.Get("X-Worker-Token"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				var payload map[string]any
				require.NoError(t, json.Unmarshal(body, &payload))
				assert.Equal(t, float64(1), payload["count"])
				return newResponse(http.StatusOK, ""), nil
			}),
		},
	}

	err := client.doRequest(context.Background(), "POST", "http://example/api/test", map[string]any{"count": 1})
	require.NoError(t, err)
}

func TestDoRequestHTTPError(t *testing.T) {
	client := &Client{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return newResponse(http.StatusBadRequest, "bad request"), nil
			}),
		},
	}

	err := client.doRequest(context.Background(), "POST", "http://example/api/test", map[string]any{"count": 1})
	var httpErr *HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusBadRequest, httpErr.StatusCode)
}

func TestDoRequestMarshalError(t *testing.T) {
	client := &Client{}
	payload := map[string]any{
		"bad": make(chan int),
	}
	err := client.doRequest(context.Background(), "POST", "http://example/api/test", payload)
	var httpErr *HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, 0, httpErr.StatusCode)
	assert.Contains(t, httpErr.Body, "marshal error")
}

func TestFetchJSONSuccess(t *testing.T) {
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() {
		pkg.Logger = prevLogger
	})

	client := &Client{
		token: "token",
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "token", r.Header.Get("X-Worker-Token"))
				assert.Equal(t, "application/json", r.Header.Get("Accept"))
				return newResponse(http.StatusOK, `{"content":"abc"}`), nil
			}),
		},
	}

	cfg, err := fetchJSON[*ProviderConfig](context.Background(), client, "http://example/api/config")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "abc", cfg.Content)
}

func TestFetchJSONStatusError(t *testing.T) {
	client := &Client{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return newResponse(http.StatusInternalServerError, "oops"), nil
			}),
		},
	}

	_, err := fetchJSON[map[string]any](context.Background(), client, "http://example/api/bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status=500")
}

func TestFetchJSONDecodeError(t *testing.T) {
	client := &Client{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return newResponse(http.StatusOK, "{bad json"), nil
			}),
		},
	}

	_, err := fetchJSON[map[string]any](context.Background(), client, "http://example/api/bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestFetchJSONRequestError(t *testing.T) {
	client := &Client{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("network down")
			}),
		},
	}

	_, err := fetchJSON[map[string]any](context.Background(), client, "http://example/api/bad")
	require.Error(t, err)
}

func TestPostBatchRoutes(t *testing.T) {
	tests := []struct {
		name       string
		dataType   string
		wantPath   string
		itemsKey   string
		otherKey   string
		otherValue string
	}{
		{
			name:     "subdomain",
			dataType: "subdomain",
			wantPath: "/api/worker/scans/1/subdomains/bulk-upsert",
			itemsKey: "subdomains",
		},
		{
			name:     "website",
			dataType: "website",
			wantPath: "/api/worker/scans/1/websites/bulk-upsert",
			itemsKey: "websites",
		},
		{
			name:     "endpoint",
			dataType: "endpoint",
			wantPath: "/api/worker/scans/1/endpoints/bulk-upsert",
			itemsKey: "endpoints",
		},
		{
			name:     "port",
			dataType: "port",
			wantPath: "/api/worker/scans/1/ports/bulk-upsert",
			itemsKey: "items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				baseURL:    "http://example",
				token:      "token",
				maxRetries: 1,
				httpClient: &http.Client{
					Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
						assert.Equal(t, tt.wantPath, r.URL.Path)
						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)
						var payload map[string]any
						require.NoError(t, json.Unmarshal(body, &payload))
						assert.Equal(t, float64(2), payload["targetId"])
						assert.Contains(t, payload, tt.itemsKey)
						return newResponse(http.StatusOK, ""), nil
					}),
				},
			}

			err := client.PostBatch(context.Background(), 1, 2, tt.dataType, []any{"a"})
			require.NoError(t, err)
		})
	}
}

func TestDoWithRetryContextCanceled(t *testing.T) {
	calls := 0
	client := &Client{
		maxRetries: 3,
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return newResponse(http.StatusOK, ""), nil
			}),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.doWithRetry(ctx, "POST", "http://example/api/test", map[string]any{"count": 1})
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, calls)
}

func TestDoWithRetryNonRetryable(t *testing.T) {
	calls := 0
	client := &Client{
		maxRetries: 3,
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return newResponse(http.StatusBadRequest, "bad"), nil
			}),
		},
	}

	err := client.doWithRetry(context.Background(), "POST", "http://example/api/test", map[string]any{"count": 1})
	require.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestDoWithRetryRetryableThenSuccess(t *testing.T) {
	calls := 0
	client := &Client{
		maxRetries: 2,
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return newResponse(http.StatusInternalServerError, "boom"), nil
				}
				return newResponse(http.StatusOK, ""), nil
			}),
		},
	}

	err := client.doWithRetry(context.Background(), "POST", "http://example/api/test", map[string]any{"count": 1})
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestDoRequestNetworkError(t *testing.T) {
	client := &Client{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("network down")
			}),
		},
	}

	err := client.doRequest(context.Background(), "POST", "http://example/api/test", map[string]any{"count": 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
}
