package domain

import "time"

type ScanID int

type ScanStatus string

const (
	ScanStatusPending   ScanStatus = "pending"
	ScanStatusRunning   ScanStatus = "running"
	ScanStatusSucceeded ScanStatus = "succeeded"
	ScanStatusFailed    ScanStatus = "failed"
	ScanStatusCancelled ScanStatus = "cancelled"
)

type Scan struct {
	ID           ScanID
	TargetID     int
	InputSource  InputSource
	TriggerType  ScanTriggerType
	Status       ScanStatus
	ErrorMessage string
	Progress     int
	CurrentStage string
	AgentID      *int
	CreatedAt    time.Time
	StoppedAt    *time.Time
}

func (scan *Scan) MarkRunning() error {
	if scan.Status != ScanStatusPending {
		return ErrInvalidStatusChange
	}
	scan.Status = ScanStatusRunning
	return nil
}

func (scan *Scan) MarkCompleted() error {
	if scan.Status != ScanStatusRunning {
		return ErrInvalidStatusChange
	}
	scan.Status = ScanStatusSucceeded
	scan.Progress = 100
	return nil
}

func (scan *Scan) MarkFailed(message string, stoppedAt time.Time) error {
	if message == "" {
		return ErrFailureMessageMissing
	}
	if scan.Status != ScanStatusPending && scan.Status != ScanStatusRunning {
		return ErrInvalidStatusChange
	}
	scan.Status = ScanStatusFailed
	scan.ErrorMessage = message
	scan.StoppedAt = &stoppedAt
	return nil
}

func (scan *Scan) Stop(stoppedAt time.Time) error {
	if !IsActiveScanStatus(scan.Status) {
		return ErrScanCannotStop
	}
	scan.Status = ScanStatusCancelled
	scan.StoppedAt = &stoppedAt
	return nil
}

func ParseScanStatus(value string) (ScanStatus, bool) {
	status := ScanStatus(value)
	switch status {
	case ScanStatusPending, ScanStatusRunning, ScanStatusSucceeded, ScanStatusFailed, ScanStatusCancelled:
		return status, true
	default:
		return "", false
	}
}

func IsActiveScanStatus(status ScanStatus) bool {
	return status == ScanStatusPending || status == ScanStatusRunning
}

func ResolveScanStatusFromTaskCounts(pending, running, failed, cancelled, skipped int) (ScanStatus, bool) {
	if pending > 0 || running > 0 {
		return "", false
	}
	if failed > 0 {
		return ScanStatusFailed, true
	}
	if cancelled > 0 {
		return ScanStatusCancelled, true
	}
	return ScanStatusSucceeded, true
}
