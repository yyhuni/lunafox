package provider

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

// WeComAdapter implements WeCom's JSON errcode acknowledgement contract.
type WeComAdapter struct {
	client *http.Client
}

// NewWeComAdapter creates a bounded WeCom webhook adapter.
func NewWeComAdapter(client *http.Client) *WeComAdapter {
	return &WeComAdapter{client: newWebhookHTTPClient(client)}
}

// Provider identifies the adapter's immutable provider protocol.
func (adapter *WeComAdapter) Provider() domain.Provider { return domain.ProviderWeCom }

// Deliver sends only a frozen markdown composition. It does not re-render any
// business payload or consult current user locale/template state.
func (adapter *WeComAdapter) Deliver(ctx context.Context, destination domain.Destination, delivery domain.Delivery) application.DeliveryResult {
	rendered, err := decodeFrozenProviderRender(delivery.RenderSnapshot)
	if err != nil {
		return application.DeliveryResult{ErrorClass: "invalid_provider_snapshot"}
	}
	payload, err := json.Marshal(struct {
		MessageType string `json:"msgtype"`
		Markdown    struct {
			Content string `json:"content"`
		} `json:"markdown"`
	}{
		MessageType: "markdown",
		Markdown: struct {
			Content string `json:"content"`
		}{Content: "**" + rendered.Title + "**\n" + rendered.Message},
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
		ErrorCode *int   `json:"errcode"`
		ErrorText string `json:"errmsg"`
	}
	if err := json.Unmarshal(response.body, &acknowledgement); err != nil || acknowledgement.ErrorCode == nil {
		return application.DeliveryResult{
			HTTPStatus:        intPointer(response.status),
			ErrorClass:        "unrecognized_provider_success",
			ProviderRequestID: response.providerRequest,
		}
	}
	if *acknowledgement.ErrorCode == 0 {
		return application.DeliveryResult{
			Accepted:          true,
			HTTPStatus:        intPointer(response.status),
			ProviderRequestID: response.providerRequest,
		}
	}
	return application.DeliveryResult{
		HTTPStatus: intPointer(response.status),
		// Provider text is untrusted and can echo request data; the stable code
		// preserves operator diagnostics without risking credential disclosure.
		ErrorClass:        "provider_business_rejected",
		ProviderRequestID: response.providerRequest,
	}
}

var _ application.ProviderAdapter = (*WeComAdapter)(nil)
