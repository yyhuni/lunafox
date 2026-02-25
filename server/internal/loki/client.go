package loki

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrLokiUnavailable = errors.New("loki unavailable")
	ErrLokiBadResponse = errors.New("loki bad response")
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type QueryRangeRequest struct {
	Query     string
	StartNs   string
	EndNs     string
	Limit     int
	Direction string
}

type StreamValue struct {
	TsNs string
	Line string
}

type StreamResult struct {
	Stream map[string]string
	Values []StreamValue
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) CheckReady(ctx context.Context) error {
	if c == nil || c.baseURL == "" {
		return fmt.Errorf("%w: empty loki url", ErrLokiUnavailable)
	}

	endpoint := c.baseURL + "/ready"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLokiUnavailable, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLokiUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: /ready status=%d", ErrLokiUnavailable, resp.StatusCode)
	}
	return nil
}

func (c *Client) QueryRange(ctx context.Context, input QueryRangeRequest) ([]StreamResult, error) {
	if c == nil || c.baseURL == "" {
		return nil, fmt.Errorf("%w: empty loki url", ErrLokiUnavailable)
	}
	if strings.TrimSpace(input.Query) == "" {
		return nil, fmt.Errorf("%w: empty query", ErrLokiBadResponse)
	}

	params := url.Values{}
	params.Set("query", input.Query)
	if input.StartNs != "" {
		params.Set("start", input.StartNs)
	}
	if input.EndNs != "" {
		params.Set("end", input.EndNs)
	}
	if input.Limit > 0 {
		params.Set("limit", strconv.Itoa(input.Limit))
	}
	if input.Direction != "" {
		params.Set("direction", strings.ToUpper(strings.TrimSpace(input.Direction)))
	}

	endpoint := c.baseURL + "/loki/api/v1/query_range?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLokiUnavailable, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLokiUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLokiBadResponse, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status=%d body=%s", ErrLokiUnavailable, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload queryRangeResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLokiBadResponse, err)
	}
	if strings.ToLower(payload.Status) != "success" {
		return nil, fmt.Errorf("%w: status=%s", ErrLokiBadResponse, payload.Status)
	}

	results := make([]StreamResult, 0, len(payload.Data.Result))
	for _, item := range payload.Data.Result {
		values := make([]StreamValue, 0, len(item.Values))
		for _, raw := range item.Values {
			if len(raw) < 2 {
				continue
			}
			values = append(values, StreamValue{
				TsNs: raw[0],
				Line: raw[1],
			})
		}
		results = append(results, StreamResult{
			Stream: item.Stream,
			Values: values,
		})
	}
	return results, nil
}

type queryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream map[string]string `json:"stream"`
			Values [][]string        `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

