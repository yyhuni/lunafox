package application

import (
	"context"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

type DestinationStore interface {
	Get(context.Context, domain.Provider) (domain.Destination, error)
	Update(context.Context, domain.Destination) (domain.Destination, error)
	DisableForWebhookRemediation(context.Context, domain.Provider) (domain.Destination, error)
	List(context.Context) ([]domain.Destination, error)
	ListEnabledForKind(context.Context, domain.Kind) ([]domain.Destination, error)
}

type DeliveryStore interface {
	Claim(context.Context, string, time.Time, time.Duration, int) ([]domain.Delivery, error)
	RecordAttempt(context.Context, domain.DeliveryAttempt, string) error
	MarkDelivered(context.Context, int64, string, time.Time) error
	MarkRetry(context.Context, int64, string, time.Time, time.Time) error
	MarkFailedTerminal(context.Context, int64, string, time.Time) error
	DeleteTerminalBefore(context.Context, domain.RetentionBatch) (int64, error)
}

// ProviderAdapter performs exactly one bounded outbound attempt from a frozen
// render snapshot and returns only redaction-safe result metadata.
type ProviderAdapter interface {
	Provider() domain.Provider
	Deliver(context.Context, domain.Destination, domain.Delivery) DeliveryResult
}

type DeliveryResult struct {
	Accepted          bool
	Retryable         bool
	RetryAfter        *time.Time
	HTTPStatus        *int
	ErrorClass        string
	ProviderRequestID string
}

// TestDeliveryResult is the only outcome shape that may cross the one-shot
// test boundary. Provider HTTP details remain inside adapters and application.
type TestDeliveryResult string

const (
	TestDeliveryDelivered           TestDeliveryResult = "delivered"
	TestDeliveryInvalidCredential   TestDeliveryResult = "invalid_credential"
	TestDeliveryConnectivityFailure TestDeliveryResult = "connectivity_failure"
	TestDeliveryProviderRejected    TestDeliveryResult = "provider_rejected"
	TestDeliveryInternalFailure     TestDeliveryResult = "internal_failure"
)
