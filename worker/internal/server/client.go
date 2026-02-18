package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"go.uber.org/zap"
)

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// IsRetryable returns true if the error should be retried
func (e *HTTPError) IsRetryable() bool {
	// 4xx errors (client errors) should not be retried
	// 5xx errors (server errors) should be retried
	return e.StatusCode >= 500
}

// isRetryableError checks if an error should be retried
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's an HTTPError
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.IsRetryable()
	}

	// Network errors (connection refused, timeout, etc.) should be retried
	// These are typically wrapped in url.Error or net.Error
	return true
}

// Client handles all HTTP communication with Server
// Implements Provider, ResultSaver, and StatusUpdater interfaces
type Client struct {
	baseURL        string
	token          string
	httpClient     *http.Client
	downloadClient *http.Client
	maxRetries     int
}

// NewClient creates a new server client
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		downloadClient: &http.Client{
			Timeout: 60 * time.Minute,
		},
		maxRetries: 3,
	}
}

// --- HTTP helpers ---

func (c *Client) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Worker-Token", c.token)
	req.Header.Set("Accept", "application/json")
	return c.httpClient.Do(req)
}

func (c *Client) getDownload(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Worker-Token", c.token)
	return c.downloadClient.Do(req)
}

func (c *Client) postWithRetry(ctx context.Context, url string, body any) error {
	return c.doWithRetry(ctx, "POST", url, body)
}

func (c *Client) doWithRetry(ctx context.Context, method, url string, body any) error {
	var lastErr error
	for i := 0; i < c.maxRetries; i++ {
		// Check context before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Exponential backoff for retries (but not on first attempt)
		if i > 0 {
			backoff := time.Duration(1<<uint(i)) * time.Second
			pkg.Logger.Info("Retrying after backoff",
				zap.String("url", url),
				zap.Int("attempt", i+1),
				zap.Duration("backoff", backoff))

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := c.doRequest(ctx, method, url, body)
		if err == nil {
			if i > 0 {
				pkg.Logger.Info("Request succeeded after retry",
					zap.String("url", url),
					zap.Int("attempts", i+1))
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			pkg.Logger.Error("Non-retryable error, aborting",
				zap.String("url", url),
				zap.Error(err))
			return err
		}

		// Log retryable error
		pkg.Logger.Warn("Retryable error occurred",
			zap.String("url", url),
			zap.Int("attempt", i+1),
			zap.Int("maxRetries", c.maxRetries),
			zap.Error(err))
	}

	pkg.Logger.Error("All retries exhausted",
		zap.String("url", url),
		zap.Int("attempts", c.maxRetries),
		zap.Error(lastErr))
	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (c *Client) doRequest(ctx context.Context, method, url string, body any) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		// JSON marshal error is not retryable (data problem)
		return &HTTPError{StatusCode: 0, Body: fmt.Sprintf("marshal error: %v", err)}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		// Request creation error is not retryable
		return &HTTPError{StatusCode: 0, Body: fmt.Sprintf("request creation error: %v", err)}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Worker-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Network errors are retryable (connection issues, timeout, etc.)
		return fmt.Errorf("network error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	return nil
}

func fetchJSON[T any](ctx context.Context, c *Client, url string) (T, error) {
	var result T

	pkg.Logger.Debug("Fetching JSON", zap.String("url", url))

	resp, err := c.get(ctx, url)
	if err != nil {
		return result, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf("server error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return result, nil
}

// PostBatch sends a batch of items to the server
func (c *Client) PostBatch(ctx context.Context, scanID, targetID int, dataType string, items []any) error {
	var url string
	var body map[string]any

	switch dataType {
	case "subdomain":
		url = fmt.Sprintf("%s/api/worker/scans/%d/subdomains/bulk-upsert", c.baseURL, scanID)
		body = map[string]any{
			"targetId":   targetID,
			"subdomains": items,
		}
	case "website":
		url = fmt.Sprintf("%s/api/worker/scans/%d/websites/bulk-upsert", c.baseURL, scanID)
		body = map[string]any{
			"targetId": targetID,
			"websites": items,
		}
	case "endpoint":
		url = fmt.Sprintf("%s/api/worker/scans/%d/endpoints/bulk-upsert", c.baseURL, scanID)
		body = map[string]any{
			"targetId":  targetID,
			"endpoints": items,
		}
	default:
		url = fmt.Sprintf("%s/api/worker/scans/%d/%ss/bulk-upsert", c.baseURL, scanID, dataType)
		body = map[string]any{
			"targetId": targetID,
			"items":    items,
		}
	}

	return c.postWithRetry(ctx, url, body)
}
