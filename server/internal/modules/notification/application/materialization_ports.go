package application

import (
	"context"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

// FactStore converges immutable facts by their stable producer event identity.
type FactStore interface {
	CreateOrGetFact(context.Context, domain.ValidatedEvent) (domain.Fact, error)
}

// AudienceStore freezes active-user recipients and their fixed inbox
// projections in the same transaction as fact materialization.
type AudienceStore interface {
	CreateRecipientsAndInbox(context.Context, domain.Fact, func(domain.Locale) (domain.RenderSnapshot, error)) (AudienceProjection, error)
}

// AudienceProjection keeps the fixed recipient and inbox projections together;
// external destination delivery remains independent from the system inbox.
type AudienceProjection struct {
	Recipients []domain.Recipient
	InboxItems []domain.InboxItem
}

// DestinationFanoutStore creates durable delivery rows for enabled exact-kind
// subscriptions without reading personal inbox preference state.
type DestinationFanoutStore interface {
	CreateDeliveries(context.Context, domain.Fact, domain.RenderSnapshot) error
}

// TemplateRenderer is server-owned and must decode only a validated envelope.
type TemplateRenderer interface {
	Render(domain.ValidatedEvent, domain.Locale) (domain.RenderSnapshot, error)
}

// RealtimeNotifier emits only after the materialization transaction commits.
type RealtimeNotifier interface {
	NotifyUser(int)
}

// MaterializationTransaction is a same-database transaction boundary. Network
// side effects must occur only after this callback commits.
type MaterializationTransaction interface {
	WithinTransaction(context.Context, func(context.Context) error) error
}
