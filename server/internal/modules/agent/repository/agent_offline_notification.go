package repository

import (
	"time"

	"gorm.io/gorm"
)

// AgentOfflineNotificationSink writes the durable candidate for an
// authoritative Agent offline transition with the caller-owned transaction.
// It must never make a network call from this producer boundary.
type AgentOfflineNotificationSink interface {
	WriteAgentOffline(tx *gorm.DB, agentID int, occurredAt time.Time) error
}
