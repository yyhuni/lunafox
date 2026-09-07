package provider

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

// DiscordAdapter implements Discord's explicit webhook acceptance contract.
type DiscordAdapter struct {
	client *http.Client
}

// NewDiscordAdapter creates a bounded Discord webhook adapter.
func NewDiscordAdapter(client *http.Client) *DiscordAdapter {
	return &DiscordAdapter{client: newWebhookHTTPClient(client)}
}

// Provider identifies the adapter's immutable provider protocol.
func (adapter *DiscordAdapter) Provider() domain.Provider { return domain.ProviderDiscord }

// Deliver sends only the frozen title/message snapshot. Business templates are
// rendered upstream and never interpreted from canonical payload here.
func (adapter *DiscordAdapter) Deliver(ctx context.Context, destination domain.Destination, delivery domain.Delivery) application.DeliveryResult {
	rendered, err := decodeFrozenProviderRender(delivery.RenderSnapshot)
	if err != nil {
		return application.DeliveryResult{ErrorClass: "invalid_provider_snapshot"}
	}
	payload, err := json.Marshal(struct {
		Content string `json:"content"`
	}{Content: rendered.Title + "\n" + rendered.Message})
	if err != nil {
		return application.DeliveryResult{ErrorClass: "provider_payload_encode"}
	}
	response, transportResult := sendWebhookJSON(ctx, adapter.client, destination.Credential, payload)
	if transportResult.ErrorClass != "" {
		return transportResult
	}
	if response.status == http.StatusNoContent {
		return application.DeliveryResult{
			Accepted:          true,
			HTTPStatus:        intPointer(response.status),
			ProviderRequestID: response.providerRequest,
		}
	}
	if response.status >= http.StatusOK && response.status < http.StatusMultipleChoices {
		return application.DeliveryResult{
			HTTPStatus:        intPointer(response.status),
			ErrorClass:        "unrecognized_provider_success",
			ProviderRequestID: response.providerRequest,
		}
	}
	return retryableHTTPResult(response)
}

var _ application.ProviderAdapter = (*DiscordAdapter)(nil)
