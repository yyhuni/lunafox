package domain

import "time"

// OutboxEvent is the persisted immutable producer envelope plus the worker
// lifecycle fields needed for lease-safe materialization.
type OutboxEvent struct {
	ID             int64
	Occurrence     Occurrence
	Status         OutboxStatus
	AvailableAt    time.Time
	LeaseOwner     string
	LeaseExpiresAt *time.Time
	FailureCode    string
	TerminalAt     *time.Time
	PublishedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
