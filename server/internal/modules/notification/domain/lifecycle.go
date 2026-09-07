package domain

// OutboxStatus is the durable materialization state of an occurrence.
type OutboxStatus string

const (
	OutboxStatusPending             OutboxStatus = "pending"
	OutboxStatusProcessing          OutboxStatus = "processing"
	OutboxStatusPublished           OutboxStatus = "published"
	OutboxStatusUnsupportedTerminal OutboxStatus = "unsupported_terminal"
)

// DeliveryStatus is the durable at-least-once provider delivery state.
type DeliveryStatus string

const (
	DeliveryStatusPending        DeliveryStatus = "pending"
	DeliveryStatusProcessing     DeliveryStatus = "processing"
	DeliveryStatusRetrying       DeliveryStatus = "retrying"
	DeliveryStatusDelivered      DeliveryStatus = "delivered"
	DeliveryStatusFailedTerminal DeliveryStatus = "failed_terminal"
)
