package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

const (
	testDeliveryTitle   = "LunaFox Notification Test"
	testDeliveryMessage = "This message verifies that your webhook configuration is working."
)

// TestDeliveryService validates one unsaved credential without writing any
// destination or durable delivery state. Its result categories are deliberately
// narrower than adapter results so external transport details cannot leak.
type TestDeliveryService struct {
	authorizer ActiveSuperuserAuthorizer
	adapters   map[domain.Provider]ProviderAdapter
}

// NewTestDeliveryService creates the authorized one-shot test use case.
func NewTestDeliveryService(authorizer ActiveSuperuserAuthorizer, adapters []ProviderAdapter) *TestDeliveryService {
	if authorizer == nil {
		panic("notification test delivery authorizer is required")
	}
	adapterMap, err := newProviderAdapterRegistry(adapters)
	if err != nil {
		panic(err)
	}
	return &TestDeliveryService{authorizer: authorizer, adapters: adapterMap}
}

// Deliver sends one fixed, data-free snapshot and never enters durable retry.
func (service *TestDeliveryService) Deliver(ctx context.Context, userID int, provider domain.Provider, credential string) (TestDeliveryResult, error) {
	if userID <= 0 {
		return "", ErrNotificationPermissionDenied
	}
	allowed, err := service.authorizer.IsActiveSuperuser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("resolve notification test delivery authority: %w", err)
	}
	if !allowed {
		return "", ErrNotificationPermissionDenied
	}
	if err := domain.ValidateDestinationCredential(provider, credential); err != nil {
		return TestDeliveryInvalidCredential, nil
	}
	adapter, found := service.adapters[provider]
	if !found {
		return "", fmt.Errorf("notification test delivery adapter %q is missing", provider)
	}
	snapshot, err := newTestDeliverySnapshot()
	if err != nil {
		return "", err
	}
	result := adapter.Deliver(context.WithoutCancel(ctx), domain.Destination{
		Provider:   provider,
		Credential: credential,
	}, domain.Delivery{Provider: provider, RenderSnapshot: snapshot})
	return classifyTestDeliveryResult(result), nil
}

func newTestDeliverySnapshot() (domain.RenderSnapshot, error) {
	payload, err := json.Marshal(map[string]any{
		"title": testDeliveryTitle, "message": testDeliveryMessage, "locale": domain.LocaleEnglish, "templateVersion": 1,
	})
	if err != nil {
		return domain.RenderSnapshot{}, fmt.Errorf("encode notification test delivery snapshot: %w", err)
	}
	return domain.RenderSnapshot{
		Locale:          domain.LocaleEnglish,
		TemplateVersion: 1,
		Title:           testDeliveryTitle,
		Message:         testDeliveryMessage,
		ProviderPayload: payload,
	}, nil
}

func classifyTestDeliveryResult(result DeliveryResult) TestDeliveryResult {
	if result.Accepted {
		return TestDeliveryDelivered
	}
	switch result.ErrorClass {
	case "timeout", "network_error", "tls_verification", "transport_error":
		return TestDeliveryConnectivityFailure
	case "provider_rejected", "provider_business_rejected", "provider_rate_limited", "unrecognized_provider_success", "response_body_too_large":
		return TestDeliveryProviderRejected
	default:
		return TestDeliveryInternalFailure
	}
}
