package provider

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

func TestDiscordAdapterRequiresExplicitNoContentAcceptance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if !strings.Contains(string(body), "Scan completed") {
			t.Fatalf("Discord payload = %s, want frozen title", body)
		}
		writer.Header().Set("X-Request-ID", "discord-request")
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	result := NewDiscordAdapter(nil).Deliver(context.Background(), testDestination(server.URL), testProviderDelivery())
	if !result.Accepted || result.HTTPStatus == nil || *result.HTTPStatus != http.StatusNoContent || result.ProviderRequestID != "discord-request" {
		t.Fatalf("Discord result = %#v, want explicit acceptance", result)
	}
}

func TestDiscordAdapterUsesThePersistedProviderSnapshotOnEveryAttempt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if !strings.Contains(string(body), "Frozen title") || !strings.Contains(string(body), "Frozen message") {
			t.Fatalf("Discord payload = %s, want persisted provider snapshot", body)
		}
		if strings.Contains(string(body), "mutated title") || strings.Contains(string(body), "mutated message") {
			t.Fatalf("Discord payload = %s, must not rebuild from mutable delivery fields", body)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	delivery := testProviderDelivery()
	delivery.RenderSnapshot.Title = "mutated title"
	delivery.RenderSnapshot.Message = "mutated message"
	delivery.RenderSnapshot.ProviderPayload = []byte(`{"title":"Frozen title","message":"Frozen message"}`)
	result := NewDiscordAdapter(nil).Deliver(context.Background(), testDestination(server.URL), delivery)
	if !result.Accepted {
		t.Fatalf("Discord result = %#v, want acceptance", result)
	}
}

func TestWeComAdapterRejectsProviderBusinessErrorInsideHTTP2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"errcode":40001,"errmsg":"invalid credential echoed nowhere"}`))
	}))
	defer server.Close()

	result := NewWeComAdapter(nil).Deliver(context.Background(), testDestination(server.URL), testProviderDelivery())
	if result.Accepted || result.Retryable || result.ErrorClass != "provider_business_rejected" {
		t.Fatalf("WeCom business rejection = %#v", result)
	}
}

func TestFeishuAdapterSendsTextPayloadAndAcceptsCodeZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if contentType := request.Header.Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", contentType)
		}
		var payload struct {
			MessageType string `json:"msg_type"`
			Content     struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode Feishu payload: %v", err)
		}
		if payload.MessageType != "text" || payload.Content.Text != "Scan completed\nScan for example.test completed successfully." {
			t.Fatalf("Feishu payload = %#v, want frozen text message", payload)
		}
		writer.Header().Set("X-Request-ID", "feishu-request")
		_, _ = writer.Write([]byte(`{"code":0}`))
	}))
	defer server.Close()

	result := NewFeishuAdapter(nil).Deliver(context.Background(), testDestination(server.URL), testProviderDelivery())
	if !result.Accepted || result.Retryable || result.HTTPStatus == nil || *result.HTTPStatus != http.StatusOK || result.ProviderRequestID != "feishu-request" {
		t.Fatalf("Feishu result = %#v, want accepted response", result)
	}
}

func TestFeishuAdapterClassifiesBusinessCodesAndHTTPResults(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		retryAfter string
		accepted   bool
		retryable  bool
		errorClass string
	}{
		{name: "code zero", status: http.StatusOK, body: `{"code":0}`, accepted: true},
		{name: "rate limited code", status: http.StatusOK, body: `{"code":11232}`, retryAfter: "30", retryable: true, errorClass: "provider_rate_limited"},
		{name: "business rejection", status: http.StatusOK, body: `{"code":19001}`, errorClass: "provider_business_rejected"},
		{name: "unrecognized success", status: http.StatusOK, body: `{"message":"missing code"}`, errorClass: "unrecognized_provider_success"},
		{name: "retryable HTTP takes precedence", status: http.StatusTooManyRequests, body: `{"code":0}`, retryable: true, errorClass: "http_retryable"},
		{name: "terminal HTTP", status: http.StatusBadRequest, body: `{"code":11232}`, errorClass: "provider_rejected"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if test.retryAfter != "" {
					writer.Header().Set("Retry-After", test.retryAfter)
				}
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()

			result := NewFeishuAdapter(nil).Deliver(context.Background(), testDestination(server.URL), testProviderDelivery())
			if result.Accepted != test.accepted || result.Retryable != test.retryable || result.ErrorClass != test.errorClass {
				t.Fatalf("Feishu result = %#v, want accepted=%t retryable=%t error=%q", result, test.accepted, test.retryable, test.errorClass)
			}
			if result.HTTPStatus == nil || *result.HTTPStatus != test.status {
				t.Fatalf("Feishu HTTP status = %#v, want %d", result.HTTPStatus, test.status)
			}
			if test.retryAfter != "" && result.RetryAfter == nil {
				t.Fatalf("Feishu retry-after = nil, want parsed value")
			}
		})
	}
}

func TestFeishuAdapterBoundsResponseBeforeBusinessParsing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Length", "65537")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(strings.Repeat("x", 65_537)))
	}))
	defer server.Close()

	result := NewFeishuAdapter(nil).Deliver(context.Background(), testDestination(server.URL), testProviderDelivery())
	if result.Accepted || result.Retryable || result.ErrorClass != "response_body_too_large" {
		t.Fatalf("Feishu oversized response = %#v", result)
	}
}

func TestWebhookAdapterBoundsResponseBeforeParsing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Length", "65537")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(strings.Repeat("x", 65_537)))
	}))
	defer server.Close()

	result := NewDiscordAdapter(nil).Deliver(context.Background(), testDestination(server.URL), testProviderDelivery())
	if result.Accepted || result.Retryable || result.ErrorClass != "response_body_too_large" {
		t.Fatalf("oversized response result = %#v", result)
	}
}

func TestWebhookAdapterRetainsRetryableStatusWhenResponseBodyIsTooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Retry-After", "30")
		writer.Header().Set("Content-Length", "65537")
		writer.WriteHeader(http.StatusTooManyRequests)
		_, _ = writer.Write([]byte(strings.Repeat("x", 65_537)))
	}))
	defer server.Close()

	result := NewDiscordAdapter(nil).Deliver(context.Background(), testDestination(server.URL), testProviderDelivery())
	if !result.Retryable || result.RetryAfter == nil || result.HTTPStatus == nil || *result.HTTPStatus != http.StatusTooManyRequests || result.ErrorClass != "response_body_too_large" {
		t.Fatalf("oversized retryable response result = %#v", result)
	}
}

func TestWebhookAdapterClassifiesRetryAfterAndTimeoutAsRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Retry-After", "30")
		writer.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	before := time.Now().UTC()
	result := NewDiscordAdapter(nil).Deliver(context.Background(), testDestination(server.URL), testProviderDelivery())
	if !result.Retryable || result.RetryAfter == nil || result.RetryAfter.Before(before.Add(29*time.Second)) {
		t.Fatalf("429 result = %#v, want retryable Retry-After", result)
	}

	timeoutClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})}
	result = NewDiscordAdapter(timeoutClient).Deliver(context.Background(), testDestination("https://discord.example.test/webhook"), testProviderDelivery())
	if !result.Retryable || result.ErrorClass != "timeout" {
		t.Fatalf("timeout result = %#v, want retryable timeout", result)
	}
}

func TestTestDeliveryUsesProviderAdaptersForFixedSnapshotAndSafeOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		provider   domain.Provider
		credential string
		adapter    application.ProviderAdapter
		want       application.TestDeliveryResult
	}{
		{
			name:       "Discord accepted",
			provider:   domain.ProviderDiscord,
			credential: "https://discord.com/api/webhooks/test-id/test-token",
			adapter: NewDiscordAdapter(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Fatalf("read Discord test body: %v", err)
				}
				if !strings.Contains(string(body), "LunaFox Notification Test") || !strings.Contains(string(body), "This message verifies that your webhook configuration is working.") {
					t.Fatalf("Discord test body = %s, want fixed English snapshot", body)
				}
				return webhookTestResponse(http.StatusNoContent, ""), nil
			})}),
			want: application.TestDeliveryDelivered,
		},
		{
			name:       "WeCom accepted",
			provider:   domain.ProviderWeCom,
			credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test-token",
			adapter: NewWeComAdapter(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Fatalf("read WeCom test body: %v", err)
				}
				if !strings.Contains(string(body), "LunaFox Notification Test") || !strings.Contains(string(body), "This message verifies that your webhook configuration is working.") {
					t.Fatalf("WeCom test body = %s, want fixed English snapshot", body)
				}
				return webhookTestResponse(http.StatusOK, `{"errcode":0}`), nil
			})}),
			want: application.TestDeliveryDelivered,
		},
		{
			name:       "Feishu accepted",
			provider:   domain.ProviderFeishu,
			credential: "https://open.feishu.cn/open-apis/bot/v2/hook/test-token",
			adapter: NewFeishuAdapter(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				var payload struct {
					MessageType string `json:"msg_type"`
					Content     struct {
						Text string `json:"text"`
					} `json:"content"`
				}
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("decode Feishu test body: %v", err)
				}
				if payload.MessageType != "text" || !strings.Contains(payload.Content.Text, "LunaFox Notification Test") || !strings.Contains(payload.Content.Text, "This message verifies that your webhook configuration is working.") {
					t.Fatalf("Feishu test body = %#v, want fixed English text snapshot", payload)
				}
				return webhookTestResponse(http.StatusOK, `{"code":0}`), nil
			})}),
			want: application.TestDeliveryDelivered,
		},
		{
			name:       "Discord provider rejected",
			provider:   domain.ProviderDiscord,
			credential: "https://discord.com/api/webhooks/test-id/test-token",
			adapter: NewDiscordAdapter(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return webhookTestResponse(http.StatusBadRequest, `{"message":"provider detail must not escape"}`), nil
			})}),
			want: application.TestDeliveryProviderRejected,
		},
		{
			name:       "WeCom provider rejected",
			provider:   domain.ProviderWeCom,
			credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test-token",
			adapter: NewWeComAdapter(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return webhookTestResponse(http.StatusOK, `{"errcode":40001,"errmsg":"provider detail must not escape"}`), nil
			})}),
			want: application.TestDeliveryProviderRejected,
		},
		{
			name:       "Feishu rate limit is terminal in one-shot test",
			provider:   domain.ProviderFeishu,
			credential: "https://open.feishu.cn/open-apis/bot/v2/hook/test-token",
			adapter: NewFeishuAdapter(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return webhookTestResponse(http.StatusOK, `{"code":11232}`), nil
			})}),
			want: application.TestDeliveryProviderRejected,
		},
		{
			name:       "timeout is connectivity failure",
			provider:   domain.ProviderDiscord,
			credential: "https://discord.com/api/webhooks/test-id/test-token",
			adapter: NewDiscordAdapter(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, context.DeadlineExceeded
			})}),
			want: application.TestDeliveryConnectivityFailure,
		},
		{
			name:       "TLS failure is connectivity failure",
			provider:   domain.ProviderWeCom,
			credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test-token",
			adapter: NewWeComAdapter(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, x509.UnknownAuthorityError{}
			})}),
			want: application.TestDeliveryConnectivityFailure,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := application.NewTestDeliveryService(webhookTestAuthorizer{}, []application.ProviderAdapter{test.adapter})
			result, err := service.Deliver(context.Background(), 7, test.provider, test.credential)
			if err != nil || result != test.want {
				t.Fatalf("Deliver() = result %q error %v, want %q", result, err, test.want)
			}
		})
	}
}

func webhookTestResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type webhookTestAuthorizer struct{}

func (webhookTestAuthorizer) IsActiveSuperuser(context.Context, int) (bool, error) {
	return true, nil
}

func testDestination(credential string) domain.Destination {
	return domain.Destination{Provider: domain.ProviderDiscord, Credential: credential, Enabled: true, Subscriptions: []domain.Kind{domain.KindScanSucceeded}}
}

func testProviderDelivery() domain.Delivery {
	return domain.Delivery{
		Provider: domain.ProviderDiscord,
		RenderSnapshot: domain.RenderSnapshot{
			Locale:          domain.LocaleEnglish,
			TemplateVersion: 1,
			Title:           "Scan completed",
			Message:         "Scan for example.test completed successfully.",
			ProviderPayload: []byte(`{"title":"Scan completed","message":"Scan for example.test completed successfully."}`),
		},
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
