package handler

import (
	"testing"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

func TestNotificationDestinationSupportedKindsExcludeInboxOnlyNucleiKinds(t *testing.T) {
	kinds := notificationSupportedKinds()
	seen := make(map[string]struct{}, len(kinds))
	for _, kind := range kinds {
		seen[kind] = struct{}{}
	}
	for _, kind := range []domain.Kind{domain.KindNucleiPOCSyncSucceeded, domain.KindNucleiPOCSyncFailed} {
		if _, found := seen[string(kind)]; found {
			t.Fatalf("destination supported kinds = %#v, unexpectedly contains inbox-only kind %q", kinds, kind)
		}
	}
	if err := domain.ValidateDestination(domain.Destination{
		Provider:      domain.ProviderDiscord,
		Credential:    "https://discord.com/api/webhooks/123/token",
		Enabled:       true,
		Subscriptions: []domain.Kind{domain.KindNucleiPOCSyncSucceeded},
	}); err == nil {
		t.Fatal("ValidateDestination accepted an inbox-only Nuclei subscription")
	}
}
