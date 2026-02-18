package task

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/agent/internal/domain"
)

const fallbackErrorSnippetRunes = 512

// Client handles HTTP API requests to the server.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewClient creates a new task client.
func NewClient(serverURL, apiKey string) *Client {
	transport := http.DefaultTransport
	if base, ok := transport.(*http.Transport); ok {
		clone := base.Clone()
		clone.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		transport = clone
	}
	return &Client{
		baseURL: strings.TrimRight(serverURL, "/"),
		apiKey:  apiKey,
		http: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
	}
}

// PullTask requests a task from the server. Returns nil when no task available.
func (c *Client) PullTask(ctx context.Context) (*domain.Task, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/agent/tasks/pull", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Agent-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pull task failed: status %d", resp.StatusCode)
	}

	var task domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateStatus reports task status to the server with retry.
func (c *Client) UpdateStatus(ctx context.Context, taskID int, status, errorMessage string) error {
	body, err := marshalStatusPayload(status, errorMessage)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(5<<attempt) * time.Second // 5s, 10s, 20s
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		statusCode, bodyText, requestErr := c.sendStatusUpdate(ctx, taskID, body)
		if requestErr != nil {
			lastErr = requestErr
			continue
		}

		if statusCode == http.StatusOK {
			return nil
		}
		if bodyText != "" {
			lastErr = fmt.Errorf("update status failed: status %d: %s", statusCode, bodyText)
		} else {
			lastErr = fmt.Errorf("update status failed: status %d", statusCode)
		}

		if shouldRetryWithCompactError(statusCode, bodyText, errorMessage) {
			fallbackMessage := compactStatusErrorMessage(errorMessage)
			fallbackBody, marshalErr := marshalStatusPayload(status, fallbackMessage)
			if marshalErr != nil {
				return marshalErr
			}

			fallbackStatusCode, fallbackBodyText, fallbackErr := c.sendStatusUpdate(ctx, taskID, fallbackBody)
			if fallbackErr != nil {
				return fmt.Errorf("%w; fallback update request failed: %v", lastErr, fallbackErr)
			}
			if fallbackStatusCode == http.StatusOK {
				return nil
			}
			if fallbackBodyText != "" {
				return fmt.Errorf("%w; fallback update failed: status %d: %s", lastErr, fallbackStatusCode, fallbackBodyText)
			}
			return fmt.Errorf("%w; fallback update failed: status %d", lastErr, fallbackStatusCode)
		}

		// Don't retry 4xx client errors (except 429)
		if statusCode >= 400 && statusCode < 500 && statusCode != 429 {
			return lastErr
		}
	}
	return lastErr
}

func marshalStatusPayload(status, errorMessage string) ([]byte, error) {
	payload := map[string]string{
		"status": status,
	}
	if errorMessage != "" {
		payload["errorMessage"] = errorMessage
	}
	return json.Marshal(payload)
}

func (c *Client) sendStatusUpdate(ctx context.Context, taskID int, body []byte) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, fmt.Sprintf("%s/api/agent/tasks/%d/status", c.baseURL, taskID), bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return resp.StatusCode, "", nil
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return resp.StatusCode, strings.TrimSpace(string(bodyBytes)), nil
}

func shouldRetryWithCompactError(statusCode int, bodyText, errorMessage string) bool {
	if statusCode != http.StatusBadRequest || errorMessage == "" {
		return false
	}
	lowerBody := strings.ToLower(bodyText)
	return strings.Contains(lowerBody, "error message exceeds 4kb limit")
}

func compactStatusErrorMessage(errorMessage string) string {
	cleaned := strings.ToValidUTF8(errorMessage, "")
	cleaned = strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			return ' '
		case r < 32 || r == 127:
			return -1
		default:
			return r
		}
	}, cleaned)
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	if cleaned == "" {
		return "worker failed (truncated): details omitted"
	}

	runes := []rune(cleaned)
	if len(runes) > fallbackErrorSnippetRunes {
		cleaned = string(runes[:fallbackErrorSnippetRunes])
	}

	return "worker failed (truncated): " + cleaned
}
