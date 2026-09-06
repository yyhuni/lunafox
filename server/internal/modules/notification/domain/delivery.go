package domain

import "time"

// Delivery is the durable, provider-independent at-least-once delivery work
// item keyed by one event and one installation destination.
type Delivery struct {
	ID             int64
	EventID        string
	FactID         int64
	DestinationID  int64
	Provider       Provider
	Status         DeliveryStatus
	AttemptCount   int
	FirstAttemptAt *time.Time
	NextAttemptAt  time.Time
	LeaseOwner     string
	LeaseExpiresAt *time.Time
	RenderSnapshot RenderSnapshot
	TerminalAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// DeliveryAttempt contains only safe provider outcome metadata. It never
// stores a credential, provider response body, or reconstructable URL secret.
type DeliveryAttempt struct {
	ID                int64
	DeliveryID        int64
	AttemptNumber     int
	AttemptedAt       time.Time
	CompletedAt       time.Time
	HTTPStatus        *int
	ErrorClass        string
	ProviderRequestID string
	CreatedAt         time.Time
}
