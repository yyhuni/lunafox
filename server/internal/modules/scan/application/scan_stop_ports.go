package application

import (
	"context"
	"time"
)

// ScanStopNotification is a committed running-task assignment eligible for a
// best-effort cancellation notification after the database transaction commits.
type ScanStopNotification struct {
	TaskID  int
	AgentID int
}

// BatchScanStopNotification carries the Scan identity because one batch may
// contain tasks assigned to different Scans.
type BatchScanStopNotification struct {
	ScanID  int
	TaskID  int
	AgentID int
}

// ScanStopOutcome is the complete committed result of a normal synchronous
// Scan Stop request. It intentionally contains no delivery state because Agent
// delivery is process-local and happens only after this outcome is durable.
type ScanStopOutcome struct {
	ScanID                 int
	CancelledTaskCount     int
	NotificationCandidates []ScanStopNotification
}

// BatchScanStopOutcome contains only state committed by one batch request.
type BatchScanStopOutcome struct {
	StoppedCount           int
	SkippedCount           int
	RevokedTaskCount       int
	NotificationCandidates []BatchScanStopNotification
}

// UpgradeScanStopOutcome is the committed result of the single maintenance
// transaction used before a host upgrade. Notification delivery is still
// best-effort and happens only after this outcome is durable.
type UpgradeScanStopOutcome struct {
	CancelledScanCount     int
	CancelledTaskCount     int
	NotificationCandidates []BatchScanStopNotification
}

// ScanStopStore owns the single transaction that revalidates and cancels an
// active Scan together with its active Tasks.
type ScanStopStore interface {
	StopActiveScan(context.Context, int, time.Time) (*ScanStopOutcome, error)
	BatchStopActiveScans(context.Context, []int, time.Time) (*BatchScanStopOutcome, error)
}

// UpgradeScanStopStore is intentionally an additive port. Existing request
// bound stop callers keep their 1-100 batch contract; the upgrade path needs a
// deployment-wide transaction and an operation identity.
type UpgradeScanStopStore interface {
	StopAllActiveScansForUpgrade(context.Context, string, time.Time) (*UpgradeScanStopOutcome, error)
}
