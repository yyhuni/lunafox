package application

import (
	"context"
	"errors"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

func TestReconcileDestinationsDisablesOnlyInvalidEnabledDestinationsAndIsIdempotent(t *testing.T) {
	legacy := domain.Destination{
		Provider:      domain.ProviderDiscord,
		Credential:    "https://discord.example.test/api/webhooks/legacy/token",
		Enabled:       true,
		Subscriptions: []domain.Kind{domain.KindScanFailed},
	}
	store := &reconciliationDestinationStore{values: []domain.Destination{
		legacy,
		{Provider: domain.ProviderWeCom, Credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=valid", Enabled: true, Subscriptions: []domain.Kind{domain.KindScanSucceeded}},
	}}
	if err := ReconcileDestinations(context.Background(), store); err != nil {
		t.Fatalf("ReconcileDestinations: %v", err)
	}
	if len(store.disabled) != 1 || store.disabled[0] != domain.ProviderDiscord {
		t.Fatalf("disabled providers = %#v", store.disabled)
	}
	updated := store.values[0]
	if updated.Enabled || updated.Credential != legacy.Credential || len(updated.Subscriptions) != 1 || updated.Subscriptions[0] != domain.KindScanFailed {
		t.Fatalf("legacy destination was not preserved during remediation: %#v", updated)
	}
	if err := ReconcileDestinations(context.Background(), store); err != nil {
		t.Fatalf("second ReconcileDestinations: %v", err)
	}
	if len(store.disabled) != 1 {
		t.Fatalf("second reconciliation changed an already disabled destination: %#v", store.disabled)
	}
}

func TestReconcileDestinationsPropagatesFailure(t *testing.T) {
	store := &reconciliationDestinationStore{err: errors.New("database unavailable")}
	if err := ReconcileDestinations(context.Background(), store); err == nil {
		t.Fatal("ReconcileDestinations() error = nil, want failure")
	}
}

type reconciliationDestinationStore struct {
	values   []domain.Destination
	disabled []domain.Provider
	err      error
}

func (store *reconciliationDestinationStore) Get(context.Context, domain.Provider) (domain.Destination, error) {
	return domain.Destination{}, store.err
}

func (*reconciliationDestinationStore) Update(context.Context, domain.Destination) (domain.Destination, error) {
	return domain.Destination{}, nil
}

func (store *reconciliationDestinationStore) DisableForWebhookRemediation(_ context.Context, provider domain.Provider) (domain.Destination, error) {
	if store.err != nil {
		return domain.Destination{}, store.err
	}
	for index := range store.values {
		if store.values[index].Provider == provider {
			store.values[index].Enabled = false
			store.disabled = append(store.disabled, provider)
			return store.values[index], nil
		}
	}
	return domain.Destination{}, errors.New("destination not found")
}

func (store *reconciliationDestinationStore) List(context.Context) ([]domain.Destination, error) {
	return append([]domain.Destination(nil), store.values...), store.err
}

func (*reconciliationDestinationStore) ListEnabledForKind(context.Context, domain.Kind) ([]domain.Destination, error) {
	return nil, nil
}
