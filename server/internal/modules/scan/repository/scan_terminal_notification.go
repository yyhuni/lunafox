package repository

import (
	"time"

	"gorm.io/gorm"
)

// ScanTerminalNotificationSink is the narrow producer seam that keeps Scan
// transition ownership here while notification envelope ownership stays out of
// the Scan module. It must use the provided transaction and perform no network
// side effect.
type ScanTerminalNotificationSink interface {
	WriteScanTerminal(tx *gorm.DB, scanID int, status, failureKind, failureMessage string, occurredAt time.Time) error
}
