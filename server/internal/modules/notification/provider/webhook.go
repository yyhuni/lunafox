// Package provider contains narrow Discord, WeCom, and Feishu transport adapters.
package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

const (
	// AttemptTimeout bounds DNS, connection, TLS, request, response read, and parsing.
	AttemptTimeout = 10 * time.Second
	// MaxResponseBodyBytes prevents untrusted provider responses from consuming worker memory.
	MaxResponseBodyBytes int64 = 65_536
)

type webhookResponse struct {
	status          int
	body            []byte
	retryAfter      *time.Time
	providerRequest string
}

type frozenProviderRender struct {
	Title           string `json:"title"`
	Message         string `json:"message"`
	Locale          string `json:"locale"`
	TemplateVersion int    `json:"templateVersion"`
}

// decodeFrozenProviderRender reads the structured render snapshot persisted with
// the delivery. Providers must not reconstruct message content from current
// templates or mutable delivery fields during a retry.
func decodeFrozenProviderRender(snapshot domain.RenderSnapshot) (frozenProviderRender, error) {
	decoder := json.NewDecoder(bytes.NewReader(snapshot.ProviderPayload))
	decoder.DisallowUnknownFields()
	var rendered frozenProviderRender
	if err := decoder.Decode(&rendered); err != nil {
		return frozenProviderRender{}, fmt.Errorf("decode frozen notification provider snapshot: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return frozenProviderRender{}, fmt.Errorf("decode frozen notification provider snapshot: multiple JSON values")
		}
		return frozenProviderRender{}, fmt.Errorf("decode frozen notification provider snapshot: %w", err)
	}
	if strings.TrimSpace(rendered.Title) == "" || strings.TrimSpace(rendered.Message) == "" {
		return frozenProviderRender{}, fmt.Errorf("frozen notification provider snapshot is missing title or message")
	}
	return rendered, nil
}

func newWebhookHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		client = &http.Client{}
	}
	copy := *client
	// Requests carry their own 10-second deadline; a client timeout could race
	// it with a less specific error classification.
	copy.Timeout = 0
	if copy.CheckRedirect == nil {
		copy.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return &copy
}

func sendWebhookJSON(ctx context.Context, client *http.Client, endpoint string, payload []byte) (webhookResponse, application.DeliveryResult) {
	attemptContext, cancel := context.WithTimeout(ctx, AttemptTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(attemptContext, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return webhookResponse{}, application.DeliveryResult{ErrorClass: "invalid_request"}
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return webhookResponse{}, classifyTransportError(err, attemptContext)
	}
	defer response.Body.Close()

	result := webhookResponse{
		status:          response.StatusCode,
		retryAfter:      parseRetryAfter(time.Now().UTC(), response.Header.Get("Retry-After")),
		providerRequest: providerRequestID(response.Header),
	}
	bodyBytes, readErr := readBoundedResponse(response)
	if readErr != nil {
		// A large body cannot be parsed as provider acceptance, but its HTTP
		// status still controls the durable retry policy. In particular, 408,
		// 429, and 5xx responses must remain retryable even when their body is
		// intentionally discarded before parsing.
		return result, application.DeliveryResult{
			Retryable:         isRetryableHTTPStatus(response.StatusCode),
			RetryAfter:        result.retryAfter,
			HTTPStatus:        intPointer(response.StatusCode),
			ErrorClass:        "response_body_too_large",
			ProviderRequestID: result.providerRequest,
		}
	}
	result.body = bodyBytes
	return result, application.DeliveryResult{HTTPStatus: intPointer(response.StatusCode), RetryAfter: result.retryAfter, ProviderRequestID: result.providerRequest}
}

func readBoundedResponse(response *http.Response) ([]byte, error) {
	if response.ContentLength > MaxResponseBodyBytes {
		return nil, fmt.Errorf("response content length %d exceeds %d", response.ContentLength, MaxResponseBodyBytes)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, MaxResponseBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > MaxResponseBodyBytes {
		return nil, fmt.Errorf("response body exceeds %d", MaxResponseBodyBytes)
	}
	return body, nil
}

func classifyTransportError(err error, attemptContext context.Context) application.DeliveryResult {
	if errors.Is(attemptContext.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return application.DeliveryResult{Retryable: true, ErrorClass: "timeout"}
	}
	var hostnameError x509.HostnameError
	var unknownAuthority x509.UnknownAuthorityError
	var certificateVerificationError *tls.CertificateVerificationError
	if errors.As(err, &hostnameError) || errors.As(err, &unknownAuthority) || errors.As(err, &certificateVerificationError) {
		return application.DeliveryResult{ErrorClass: "tls_verification"}
	}
	var urlError *url.Error
	if errors.As(err, &urlError) && strings.Contains(strings.ToLower(urlError.Err.Error()), "certificate") {
		return application.DeliveryResult{ErrorClass: "tls_verification"}
	}
	var networkError net.Error
	if errors.As(err, &networkError) || errors.Is(err, context.Canceled) {
		return application.DeliveryResult{Retryable: true, ErrorClass: "network_error"}
	}
	// A parsed destination URL has already passed deterministic validation;
	// unknown client transport errors are still safe to retry under the budget.
	return application.DeliveryResult{Retryable: true, ErrorClass: "transport_error"}
}

func retryableHTTPResult(response webhookResponse) application.DeliveryResult {
	if isRetryableHTTPStatus(response.status) {
		return application.DeliveryResult{
			Retryable:         true,
			RetryAfter:        response.retryAfter,
			HTTPStatus:        intPointer(response.status),
			ErrorClass:        "http_retryable",
			ProviderRequestID: response.providerRequest,
		}
	}
	return application.DeliveryResult{
		HTTPStatus:        intPointer(response.status),
		ErrorClass:        "provider_rejected",
		ProviderRequestID: response.providerRequest,
	}
}

func isRetryableHTTPStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

func parseRetryAfter(now time.Time, raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil && seconds >= 0 {
		value := now.Add(time.Duration(seconds) * time.Second)
		return &value
	}
	if value, err := http.ParseTime(raw); err == nil && value.After(now) {
		value = value.UTC()
		return &value
	}
	return nil
}

func providerRequestID(header http.Header) string {
	for _, key := range []string{"X-Request-ID", "X-Request-Id", "X-Trace-ID"} {
		if value := strings.TrimSpace(header.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func intPointer(value int) *int { return &value }
