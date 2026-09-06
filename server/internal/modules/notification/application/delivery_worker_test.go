package application

import (
	"context"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

func TestDeliveryWorkerPersistsRetryWithFullJitterSchedule(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	store := &deliveryStoreFake{claimed: []domain.Delivery{testDelivery(now, 0)}}
	destination := &deliveryDestinationStoreFake{destination: testEnabledDestination()}
	adapter := &deliveryAdapterFake{result: DeliveryResult{Retryable: true, ErrorClass: "timeout"}}
	worker := NewDeliveryWorker(store, destination, []ProviderAdapter{adapter}, DeliveryWorkerOptions{Owner: "worker"})
	worker.now = func() time.Time { return now }
	worker.jitter = func(cap time.Duration) time.Duration {
		if cap != initialRetryBackoff {
			t.Fatalf("retry cap = %s, want %s", cap, initialRetryBackoff)
		}
		return 12 * time.Second
	}
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(store.attempts) != 1 || store.attempts[0].AttemptNumber != 1 || store.attempts[0].ErrorClass != "timeout" {
		t.Fatalf("attempts = %#v", store.attempts)
	}
	if len(store.retries) != 1 || !store.retries[0].next.Equal(now.Add(12*time.Second)) {
		t.Fatalf("retries = %#v, want next at %s", store.retries, now.Add(12*time.Second))
	}
	if len(store.delivered) != 0 || len(store.failed) != 0 {
		t.Fatalf("retryable result terminally transitioned: delivered=%#v failed=%#v", store.delivered, store.failed)
	}
}

func TestDeliveryWorkerTerminalizesAtSixthAttempt(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	delivery := testDelivery(now, 5)
	delivery.FirstAttemptAt = &now
	store := &deliveryStoreFake{claimed: []domain.Delivery{delivery}}
	adapter := &deliveryAdapterFake{result: DeliveryResult{Retryable: true, ErrorClass: "http_retryable"}}
	worker := NewDeliveryWorker(store, &deliveryDestinationStoreFake{destination: testEnabledDestination()}, []ProviderAdapter{adapter}, DeliveryWorkerOptions{Owner: "worker"})
	worker.now = func() time.Time { return now }
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(store.failed) != 1 || store.failed[0].id != delivery.ID {
		t.Fatalf("terminal transitions = %#v", store.failed)
	}
	if len(store.retries) != 0 {
		t.Fatalf("sixth attempt scheduled retry: %#v", store.retries)
	}
}

func TestDeliveryWorkerMarksProviderAcceptanceDelivered(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	store := &deliveryStoreFake{claimed: []domain.Delivery{testDelivery(now, 0)}}
	adapter := &deliveryAdapterFake{result: DeliveryResult{Accepted: true, HTTPStatus: intPointerForTest(204)}}
	worker := NewDeliveryWorker(store, &deliveryDestinationStoreFake{destination: testEnabledDestination()}, []ProviderAdapter{adapter}, DeliveryWorkerOptions{Owner: "worker"})
	worker.now = func() time.Time { return now }
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(store.delivered) != 1 || store.delivered[0].id != 1 {
		t.Fatalf("delivered transitions = %#v", store.delivered)
	}
}

func TestDeliveryWorkerRejectsDisabledDestinationWithoutNetworkAttempt(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	store := &deliveryStoreFake{claimed: []domain.Delivery{testDelivery(now, 0)}}
	adapter := &deliveryAdapterFake{result: DeliveryResult{Accepted: true}}
	destination := testEnabledDestination()
	destination.Enabled = false
	worker := NewDeliveryWorker(store, &deliveryDestinationStoreFake{destination: destination}, []ProviderAdapter{adapter}, DeliveryWorkerOptions{Owner: "worker"})
	worker.now = func() time.Time { return now }
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if adapter.calls != 0 || len(store.attempts) != 0 || len(store.failed) != 1 {
		t.Fatalf("disabled destination result: adapter=%d attempts=%#v failed=%#v", adapter.calls, store.attempts, store.failed)
	}
}

func TestDeliveryWorkerRejectsNoncompliantDestinationWithoutNetworkAttempt(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	store := &deliveryStoreFake{claimed: []domain.Delivery{testDelivery(now, 0)}}
	adapter := &deliveryAdapterFake{result: DeliveryResult{Accepted: true}}
	destination := testEnabledDestination()
	destination.Credential = "https://discord.example.test/api/webhooks/legacy/token"
	worker := NewDeliveryWorker(store, &deliveryDestinationStoreFake{destination: destination}, []ProviderAdapter{adapter}, DeliveryWorkerOptions{Owner: "worker"})
	worker.now = func() time.Time { return now }
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if adapter.calls != 0 || len(store.attempts) != 0 || len(store.failed) != 1 {
		t.Fatalf("noncompliant destination result: adapter=%d attempts=%#v failed=%#v", adapter.calls, store.attempts, store.failed)
	}
}

func TestDeliveryWorkerRetriesFeishuRateLimitFromProviderBusinessCode(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	delivery := testDelivery(now, 0)
	delivery.Provider = domain.ProviderFeishu
	destination := testEnabledDestination()
	destination.Provider = domain.ProviderFeishu
	destination.Credential = "https://open.feishu.cn/open-apis/bot/v2/hook/worker-token"
	retryAt := now.Add(30 * time.Second)
	store := &deliveryStoreFake{claimed: []domain.Delivery{delivery}}
	adapter := &deliveryAdapterFake{provider: domain.ProviderFeishu, result: DeliveryResult{
		Retryable:  true,
		RetryAfter: &retryAt,
		ErrorClass: "provider_rate_limited",
	}}
	worker := NewDeliveryWorker(store, &deliveryDestinationStoreFake{destination: destination}, []ProviderAdapter{adapter}, DeliveryWorkerOptions{Owner: "worker"})
	worker.now = func() time.Time { return now }
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if adapter.calls != 1 || len(store.retries) != 1 || !store.retries[0].next.Equal(retryAt) {
		t.Fatalf("Feishu rate-limit transition: calls=%d retries=%#v", adapter.calls, store.retries)
	}
	if len(store.failed) != 0 || len(store.delivered) != 0 {
		t.Fatalf("Feishu rate-limit became terminal: failed=%#v delivered=%#v", store.failed, store.delivered)
	}
}

type deliveryStoreFake struct {
	claimed   []domain.Delivery
	attempts  []domain.DeliveryAttempt
	delivered []deliveryTransition
	retries   []deliveryRetry
	failed    []deliveryTransition
}

type deliveryTransition struct {
	id    int64
	owner string
}

type deliveryRetry struct {
	id    int64
	owner string
	next  time.Time
}

func (store *deliveryStoreFake) Claim(context.Context, string, time.Time, time.Duration, int) ([]domain.Delivery, error) {
	claimed := store.claimed
	store.claimed = nil
	return claimed, nil
}

func (store *deliveryStoreFake) RecordAttempt(_ context.Context, attempt domain.DeliveryAttempt, _ string) error {
	store.attempts = append(store.attempts, attempt)
	return nil
}

func (store *deliveryStoreFake) MarkDelivered(_ context.Context, id int64, owner string, _ time.Time) error {
	store.delivered = append(store.delivered, deliveryTransition{id: id, owner: owner})
	return nil
}

func (store *deliveryStoreFake) MarkRetry(_ context.Context, id int64, owner string, next, _ time.Time) error {
	store.retries = append(store.retries, deliveryRetry{id: id, owner: owner, next: next})
	return nil
}

func (store *deliveryStoreFake) MarkFailedTerminal(_ context.Context, id int64, owner string, _ time.Time) error {
	store.failed = append(store.failed, deliveryTransition{id: id, owner: owner})
	return nil
}

func (*deliveryStoreFake) DeleteTerminalBefore(context.Context, domain.RetentionBatch) (int64, error) {
	return 0, nil
}

type deliveryDestinationStoreFake struct {
	destination domain.Destination
	err         error
}

func (store *deliveryDestinationStoreFake) Get(context.Context, domain.Provider) (domain.Destination, error) {
	return store.destination, store.err
}

func (*deliveryDestinationStoreFake) Update(context.Context, domain.Destination) (domain.Destination, error) {
	return domain.Destination{}, nil
}

func (store *deliveryDestinationStoreFake) DisableForWebhookRemediation(_ context.Context, provider domain.Provider) (domain.Destination, error) {
	if store.destination.Provider != provider {
		return domain.Destination{}, store.err
	}
	store.destination.Enabled = false
	return store.destination, store.err
}

func (store *deliveryDestinationStoreFake) List(context.Context) ([]domain.Destination, error) {
	return []domain.Destination{store.destination}, store.err
}

func (*deliveryDestinationStoreFake) ListEnabledForKind(context.Context, domain.Kind) ([]domain.Destination, error) {
	return nil, nil
}

type deliveryAdapterFake struct {
	provider domain.Provider
	result   DeliveryResult
	calls    int
}

func (adapter *deliveryAdapterFake) Provider() domain.Provider {
	if adapter.provider != "" {
		return adapter.provider
	}
	return domain.ProviderDiscord
}

func (adapter *deliveryAdapterFake) Deliver(context.Context, domain.Destination, domain.Delivery) DeliveryResult {
	adapter.calls++
	return adapter.result
}

func testDelivery(now time.Time, attemptCount int) domain.Delivery {
	return domain.Delivery{
		ID:            1,
		EventID:       "scan:1:succeeded",
		FactID:        9,
		DestinationID: 3,
		Provider:      domain.ProviderDiscord,
		Status:        domain.DeliveryStatusProcessing,
		AttemptCount:  attemptCount,
		RenderSnapshot: domain.RenderSnapshot{
			Locale:          domain.LocaleEnglish,
			TemplateVersion: 1,
			Title:           "Scan completed",
			Message:         "Completed",
			ProviderPayload: []byte(`{"title":"Scan completed","message":"Completed"}`),
		},
		NextAttemptAt: now,
	}
}

func testEnabledDestination() domain.Destination {
	return domain.Destination{
		ID:            3,
		Provider:      domain.ProviderDiscord,
		Credential:    "https://discord.com/api/webhooks/worker/token",
		Enabled:       true,
		Subscriptions: []domain.Kind{domain.KindScanSucceeded},
	}
}

func intPointerForTest(value int) *int { return &value }
