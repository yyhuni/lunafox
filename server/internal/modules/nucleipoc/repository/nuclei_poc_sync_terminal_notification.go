package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NucleiPOCSyncTerminalNotificationSink is the narrow producer seam that
// keeps Nuclei terminal-state ownership in this repository while notification
// envelope construction stays in the notification module. Implementations
// must use the caller-owned transaction and perform no network side effect.
type NucleiPOCSyncTerminalNotificationSink interface {
	WriteNucleiPOCSyncSucceeded(tx *gorm.DB, taskID uuid.UUID, occurredAt time.Time) error
	WriteNucleiPOCSyncFailed(tx *gorm.DB, taskID uuid.UUID, occurredAt time.Time) error
}
