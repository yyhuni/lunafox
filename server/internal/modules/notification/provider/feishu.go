package provider

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

// FeishuAdapter implements Feishu custom-bot text message delivery.
type FeishuAdapter struct {
	client *http.Client
}

// NewFeishuAdapter creates a bounded Feishu custom-bot webhook adapter.
func NewFeishuAdapter(client *http.Client) *FeishuAdapter {
	return &FeishuAdapter{client: newWebhookHTTPClient(client)}
}

// Provider identifies the adapter's immutable provider protocol.
func (adapter *FeishuAdapter) Provider() domain.Provider { return domain.ProviderFeishu }

// Deliver sends one ordinary Feishu text payload built from the frozen render
// snapshot. The adapter never selects a template or reads mutable delivery
// content while retrying an already materialized notification.
func (adapter *FeishuAdapter) Deliver(ctx context.Context, destination domain.Destination, delivery domain.Delivery) application.DeliveryResult {
	rendered, err := decodeFrozenProviderRender(delivery.RenderSnapshot)
	if err != nil {
		return application.DeliveryResult{ErrorClass: "invalid_provider_snapshot"}
	}
	payload, err := json.Marshal(struct {
		MessageType string `json:"msg_type"`
		Content     struct {
			Text string `json:"text"`
		} `json:"content"`
	}{
		MessageType: "text",
		Content: struct {
			Text string `json:"text"`
		}{Text: rendered.Title + "\n" + rendered.Message},
	})
	if err != nil {
		return application.DeliveryResult{ErrorClass: "provider_payload_encode"}
	}
	response, transportResult := sendWebhookJSON(ctx, adapter.client, destination.Credential, payload)
	if transportResult.ErrorClass != "" {
		return transportResult
	}
	if response.status < http.StatusOK || response.status >= http.StatusMultipleChoices {
		return retryableHTTPResult(response)
	}
	var acknowledgement struct {
		Code *int `json:"code"`
	}
	if err := json.Unmarshal(response.body, &acknowledgement); err != nil || acknowledgement.Code == nil {
		return application.DeliveryResult{
			HTTPStatus:        intPointer(response.status),
			ErrorClass:        "unrecognized_provider_success",
			ProviderRequestID: response.providerRequest,
		}
	}
	switch *acknowledgement.Code {
	case 0:
		return application.DeliveryResult{
			Accepted:          true,
			HTTPStatus:        intPointer(response.status),
			ProviderRequestID: response.providerRequest,
		}
	case 11232:
		return application.DeliveryResult{
			Retryable:         true,
			RetryAfter:        response.retryAfter,
			HTTPStatus:        intPointer(response.status),
			ErrorClass:        "provider_rate_limited",
			ProviderRequestID: response.providerRequest,
		}
	default:
		return application.DeliveryResult{
			HTTPStatus:        intPointer(response.status),
			ErrorClass:        "provider_business_rejected",
			ProviderRequestID: response.providerRequest,
		}
	}
}

var _ application.ProviderAdapter = (*FeishuAdapter)(nil)
