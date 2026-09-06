package geolocation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	freeIPAPIEndpoint        = "https://free.freeipapi.com/api/json"
	freeIPAPIAttemptTimeout  = 3 * time.Second
	freeIPAPIMaxBodyBytes    = 64 << 10
	defaultRateLimitCooldown = time.Minute
)

var errInsecureFreeIPAPIRedirect = errors.New("FreeIPAPI redirect must use HTTPS")

// FreeIPAPIClient is the fixed credential-free FreeIPAPI adapter.
type FreeIPAPIClient struct {
	client   *http.Client
	endpoint string
	timeout  time.Duration
	now      func() time.Time
}

// NewFreeIPAPIClient constructs the fixed production adapter without provider credentials.
func NewFreeIPAPIClient(client *http.Client) *FreeIPAPIClient {
	if client == nil {
		client = http.DefaultClient
	}
	clientCopy := *client
	clientCopy.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if request.URL.Scheme != "https" {
			return errInsecureFreeIPAPIRedirect
		}
		if len(via) >= 10 {
			return errors.New("FreeIPAPI redirect limit exceeded")
		}
		return nil
	}
	return &FreeIPAPIClient{
		client:   &clientCopy,
		endpoint: freeIPAPIEndpoint,
		timeout:  freeIPAPIAttemptTimeout,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// Lookup performs one logical target-IP lookup, or a no-target self lookup when targetIP is empty.
func (client *FreeIPAPIClient) Lookup(ctx context.Context, targetIP string) (Location, error) {
	if ctx == nil || client == nil || client.client == nil || client.now == nil || client.timeout <= 0 {
		return Location{}, &LookupError{Class: FailureInvalidTarget}
	}
	normalizedTarget := ""
	if strings.TrimSpace(targetIP) != "" {
		var ok bool
		normalizedTarget, ok = NormalizePublicIP(targetIP)
		if !ok {
			return Location{}, &LookupError{Class: FailureInvalidTarget}
		}
	}

	attemptContext, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(attemptContext, http.MethodGet, client.lookupURL(normalizedTarget), nil)
	if err != nil {
		return Location{}, &LookupError{Class: FailureInvalidTarget}
	}
	response, err := client.client.Do(request)
	if err != nil {
		return Location{}, classifyFreeIPAPITransportError(attemptContext, err)
	}
	defer response.Body.Close()

	attemptedAt := client.now().UTC()
	if response.StatusCode == http.StatusTooManyRequests {
		return Location{}, &LookupError{
			Class:        FailureRateLimited,
			RetryAfterAt: parseRetryAfter(response.Header.Get("Retry-After"), attemptedAt),
		}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return Location{}, &LookupError{Class: FailureHTTPStatus}
	}

	payload, err := io.ReadAll(io.LimitReader(response.Body, freeIPAPIMaxBodyBytes+1))
	if err != nil {
		if attemptContext.Err() != nil {
			return Location{}, classifyFreeIPAPITransportError(attemptContext, attemptContext.Err())
		}
		return Location{}, &LookupError{Class: FailureNetwork}
	}
	if len(payload) > freeIPAPIMaxBodyBytes {
		return Location{}, &LookupError{Class: FailureBodyTooLarge}
	}
	location, err := decodeFreeIPAPILocation(payload, normalizedTarget, attemptedAt)
	if err != nil {
		return Location{}, &LookupError{Class: FailureInvalidPayload}
	}
	return location, nil
}

func (client *FreeIPAPIClient) lookupURL(normalizedTarget string) string {
	if normalizedTarget == "" {
		return client.endpoint
	}
	return strings.TrimRight(client.endpoint, "/") + "/" + url.PathEscape(normalizedTarget)
}

func classifyFreeIPAPITransportError(ctx context.Context, err error) *LookupError {
	if errors.Is(err, errInsecureFreeIPAPIRedirect) || strings.Contains(err.Error(), "redirect limit exceeded") {
		return &LookupError{Class: FailureRedirect}
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return &LookupError{Class: FailureCanceled}
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return &LookupError{Class: FailureTimeout}
	}
	return &LookupError{Class: FailureNetwork}
}

func decodeFreeIPAPILocation(payload []byte, targetIP string, resolvedAt time.Time) (Location, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	var fields map[string]json.RawMessage
	if err := decoder.Decode(&fields); err != nil {
		return Location{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Location{}, errors.New("FreeIPAPI response contains trailing JSON")
	}

	observedIP, err := requiredJSONString(fields, "ipAddress")
	if err != nil {
		return Location{}, err
	}
	observedIP, ok := NormalizePublicIP(observedIP)
	if !ok || targetIP != "" && observedIP != targetIP {
		return Location{}, errors.New("FreeIPAPI response IP is invalid")
	}
	latitude, err := requiredJSONFloat(fields, "latitude")
	if err != nil || !finiteInRange(latitude, -90, 90) {
		return Location{}, errors.New("FreeIPAPI latitude is invalid")
	}
	longitude, err := requiredJSONFloat(fields, "longitude")
	if err != nil || !finiteInRange(longitude, -180, 180) {
		return Location{}, errors.New("FreeIPAPI longitude is invalid")
	}
	radius, err := nullableJSONFloat(fields, "accuracyRadiusKm")
	if err != nil || radius != nil && (math.IsNaN(*radius) || math.IsInf(*radius, 0) || *radius < 0) {
		return Location{}, errors.New("FreeIPAPI accuracy radius is invalid")
	}

	return Location{
		ObservedIP:       observedIP,
		Latitude:         latitude,
		Longitude:        longitude,
		AccuracyRadiusKM: radius,
		ProviderKey:      FreeIPAPIProviderKey,
		ResolvedAt:       resolvedAt.UTC(),
	}, nil
}

func requiredJSONString(fields map[string]json.RawMessage, name string) (string, error) {
	raw, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", fmt.Errorf("%s is required", name)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s must be a non-empty string", name)
	}
	return value, nil
}

func requiredJSONFloat(fields map[string]json.RawMessage, name string) (float64, error) {
	raw, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return 0, fmt.Errorf("%s is required", name)
	}
	var value float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, fmt.Errorf("%s must be numeric", name)
	}
	return value, nil
}

func nullableJSONFloat(fields map[string]json.RawMessage, name string) (*float64, error) {
	raw, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	value, err := requiredJSONFloat(fields, name)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func finiteInRange(value, minimum, maximum float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= minimum && value <= maximum
}

func parseRetryAfter(raw string, now time.Time) time.Time {
	trimmed := strings.TrimSpace(raw)
	if seconds, err := strconv.ParseInt(trimmed, 10, 64); err == nil && seconds > 0 {
		return now.Add(time.Duration(seconds) * time.Second)
	}
	if parsed, err := http.ParseTime(trimmed); err == nil && parsed.After(now) {
		return parsed.UTC()
	}
	return now.Add(defaultRateLimitCooldown)
}
