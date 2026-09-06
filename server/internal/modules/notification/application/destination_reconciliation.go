package application

import (
	"context"
	"fmt"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

// ReconcileDestinations disables only enabled legacy destinations that violate
// the current outbound endpoint policy. Retaining their credential and
// subscriptions lets an authorized operator repair the configuration.
func ReconcileDestinations(ctx context.Context, destinations DestinationStore) error {
	if destinations == nil {
		panic("notification destination reconciliation store is required")
	}
	values, err := destinations.List(ctx)
	if err != nil {
		return fmt.Errorf("list notification destinations for reconciliation: %w", err)
	}
	for _, destination := range values {
		if !destination.Enabled || !domain.RequiresWebhookUpdate(destination) {
			continue
		}
		if _, err := destinations.DisableForWebhookRemediation(ctx, destination.Provider); err != nil {
			return fmt.Errorf("disable invalid notification destination %q: %w", destination.Provider, err)
		}
	}
	return nil
}
