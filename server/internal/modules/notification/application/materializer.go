package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

// Materializer converts one claimed immutable envelope into durable facts,
// frozen recipients/inbox rows, and delivery work before sending refresh hints.
type Materializer struct {
	transaction  MaterializationTransaction
	facts        FactStore
	audience     AudienceStore
	destinations DestinationFanoutStore
	renderer     TemplateRenderer
	notifier     RealtimeNotifier
}

func NewMaterializer(transaction MaterializationTransaction, facts FactStore, audience AudienceStore, destinations DestinationFanoutStore, renderer TemplateRenderer, notifier RealtimeNotifier) *Materializer {
	if transaction == nil || facts == nil || audience == nil || destinations == nil || renderer == nil {
		panic("notification materializer dependencies are required")
	}
	return &Materializer{transaction: transaction, facts: facts, audience: audience, destinations: destinations, renderer: renderer, notifier: notifier}
}

func (materializer *Materializer) Materialize(ctx context.Context, event domain.OutboxEvent) error {
	if err := domain.ValidateOccurrence(event.Occurrence); err != nil {
		return &UnsupportedEventError{Code: "invalid_envelope", Err: err}
	}
	validated, err := domain.DecodeOccurrence(event.Occurrence)
	if err != nil {
		return &UnsupportedEventError{Code: "unsupported_envelope", Err: err}
	}

	var projection AudienceProjection
	err = materializer.transaction.WithinTransaction(ctx, func(txContext context.Context) error {
		fact, err := materializer.facts.CreateOrGetFact(txContext, validated)
		if err != nil {
			return err
		}
		projection, err = materializer.audience.CreateRecipientsAndInbox(txContext, fact, func(locale domain.Locale) (domain.RenderSnapshot, error) {
			snapshot, renderErr := materializer.renderer.Render(validated, locale)
			if renderErr != nil {
				return domain.RenderSnapshot{}, &UnsupportedEventError{Code: "template_render_failed", Err: renderErr}
			}
			return snapshot, nil
		})
		if err != nil {
			return err
		}
		locale := domain.LocaleEnglish
		if len(projection.Recipients) > 0 {
			locale = projection.Recipients[0].Locale
		}
		snapshot, err := materializer.renderer.Render(validated, locale)
		if err != nil {
			return &UnsupportedEventError{Code: "template_render_failed", Err: err}
		}
		return materializer.destinations.CreateDeliveries(txContext, fact, snapshot)
	})
	if err != nil {
		var unsupported *UnsupportedEventError
		if errors.As(err, &unsupported) {
			return err
		}
		var deterministic *domain.DeterministicError
		if errors.As(err, &deterministic) {
			code := deterministic.Code
			if code == "" {
				code = "deterministic_materialization_error"
			}
			return &UnsupportedEventError{Code: code, Err: err}
		}
		return err
	}
	if materializer.notifier != nil {
		seen := make(map[int]struct{}, len(projection.InboxItems))
		for _, item := range projection.InboxItems {
			if _, ok := seen[item.UserID]; ok {
				continue
			}
			seen[item.UserID] = struct{}{}
			materializer.notifier.NotifyUser(item.UserID)
		}
	}
	return nil
}

var _ OutboxMaterializer = (*Materializer)(nil)

func unsupportedError(code string, err error) error {
	return &UnsupportedEventError{Code: code, Err: fmt.Errorf("%s: %w", code, err)}
}
