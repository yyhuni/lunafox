package application

import (
	"context"
	"errors"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

func TestTestDeliveryUsesOneFixedSnapshotWithoutDurableState(t *testing.T) {
	adapter := &testDeliveryAdapter{provider: domain.ProviderDiscord, result: DeliveryResult{Accepted: true}}
	service := NewTestDeliveryService(testDeliveryAuthorizer{allowed: true}, []ProviderAdapter{adapter})

	result, err := service.Deliver(context.Background(), 7, domain.ProviderDiscord, "https://discord.com/api/webhooks/test-id/test-token")
	if err != nil || result != TestDeliveryDelivered {
		t.Fatalf("Deliver() = result %q error %v, want delivered", result, err)
	}
	if adapter.calls != 1 {
		t.Fatalf("adapter calls = %d, want 1", adapter.calls)
	}
	if adapter.destination.Enabled || len(adapter.destination.Subscriptions) != 0 {
		t.Fatalf("test destination unexpectedly carries persisted state: %#v", adapter.destination)
	}
	if adapter.delivery.RenderSnapshot.Locale != domain.LocaleEnglish ||
		adapter.delivery.RenderSnapshot.Title != testDeliveryTitle ||
		adapter.delivery.RenderSnapshot.Message != testDeliveryMessage {
		t.Fatalf("test snapshot = %#v", adapter.delivery.RenderSnapshot)
	}
}

func TestTestDeliveryRejectsUnauthorizedAndInvalidCredentialsBeforeAdapter(t *testing.T) {
	adapter := &testDeliveryAdapter{provider: domain.ProviderDiscord}
	unauthorized := NewTestDeliveryService(testDeliveryAuthorizer{allowed: false}, []ProviderAdapter{adapter})
	if _, err := unauthorized.Deliver(context.Background(), 7, domain.ProviderDiscord, "https://discord.com/api/webhooks/test-id/test-token"); !errors.Is(err, ErrNotificationPermissionDenied) {
		t.Fatalf("unauthorized error = %v, want permission denied", err)
	}
	service := NewTestDeliveryService(testDeliveryAuthorizer{allowed: true}, []ProviderAdapter{adapter})
	result, err := service.Deliver(context.Background(), 7, domain.ProviderDiscord, "https://discord.example.test/api/webhooks/test-id/test-token")
	if err != nil || result != TestDeliveryInvalidCredential {
		t.Fatalf("invalid credential result = %q error %v", result, err)
	}
	if adapter.calls != 0 {
		t.Fatalf("adapter calls = %d, want no outbound attempt", adapter.calls)
	}
}

func TestTestDeliveryClassifiesOnlySafeTerminalOutcomes(t *testing.T) {
	tests := []struct {
		name   string
		result DeliveryResult
		want   TestDeliveryResult
	}{
		{name: "connectivity", result: DeliveryResult{ErrorClass: "timeout"}, want: TestDeliveryConnectivityFailure},
		{name: "provider rejected", result: DeliveryResult{ErrorClass: "provider_rejected", HTTPStatus: intPointerForTest(400), ProviderRequestID: "must-not-leak"}, want: TestDeliveryProviderRejected},
		{name: "internal", result: DeliveryResult{ErrorClass: "provider_payload_encode"}, want: TestDeliveryInternalFailure},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			adapter := &testDeliveryAdapter{provider: domain.ProviderDiscord, result: test.result}
			service := NewTestDeliveryService(testDeliveryAuthorizer{allowed: true}, []ProviderAdapter{adapter})
			got, err := service.Deliver(context.Background(), 7, domain.ProviderDiscord, "https://discord.com/api/webhooks/test-id/test-token")
			if err != nil || got != test.want {
				t.Fatalf("Deliver() = result %q error %v, want %q", got, err, test.want)
			}
		})
	}
}

func TestValidateFixedProviderAdaptersRequiresFeishuPreflight(t *testing.T) {
	discord := &testDeliveryAdapter{provider: domain.ProviderDiscord}
	wecom := &testDeliveryAdapter{provider: domain.ProviderWeCom}
	feishu := &testDeliveryAdapter{provider: domain.ProviderFeishu}
	if err := ValidateFixedProviderAdapters([]ProviderAdapter{discord, wecom}); err == nil {
		t.Fatal("ValidateFixedProviderAdapters() error = nil, want missing Feishu adapter")
	}
	if err := ValidateFixedProviderAdapters([]ProviderAdapter{discord, wecom, feishu}); err != nil {
		t.Fatalf("ValidateFixedProviderAdapters() error = %v", err)
	}
}

type testDeliveryAuthorizer struct {
	allowed bool
	err     error
}

func (authorizer testDeliveryAuthorizer) IsActiveSuperuser(context.Context, int) (bool, error) {
	return authorizer.allowed, authorizer.err
}

type testDeliveryAdapter struct {
	provider    domain.Provider
	result      DeliveryResult
	calls       int
	destination domain.Destination
	delivery    domain.Delivery
}

func (adapter *testDeliveryAdapter) Provider() domain.Provider { return adapter.provider }

func (adapter *testDeliveryAdapter) Deliver(_ context.Context, destination domain.Destination, delivery domain.Delivery) DeliveryResult {
	adapter.calls++
	adapter.destination = destination
	adapter.delivery = delivery
	return adapter.result
}
