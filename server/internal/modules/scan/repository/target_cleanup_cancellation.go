package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TargetCleanupTaskCancelCandidate identifies a running Task whose assignment
// was complete before its cancellation committed. It is intentionally only the
// data needed for a best-effort post-commit Agent notification.
type TargetCleanupTaskCancelCandidate struct {
	TaskID  int
	AgentID int
}

// TargetCleanupScanCancellation is the committed cancellation outcome for one
// active Scan. Terminal Tasks are deliberately excluded from both counts and
// notification candidates.
type TargetCleanupScanCancellation struct {
	ScanID                 int
	CancelledTaskCount     int
	NotificationCandidates []TargetCleanupTaskCancelCandidate
}

type targetCleanupScanRow struct {
	ID int `gorm:"column:id"`
}

type targetCleanupTaskRow struct {
	ID                   int     `gorm:"column:id"`
	Status               string  `gorm:"column:status"`
	AssignedAgentID      *int    `gorm:"column:assigned_agent_id"`
	AssignedSessionID    *string `gorm:"column:assigned_session_id"`
	AssignedSessionEpoch *int64  `gorm:"column:assigned_session_epoch"`
	AssignedRequestID    *string `gorm:"column:assigned_request_id"`
}

// CancelNextActiveForDeletedTarget cancels at most one Scan in one transaction.
// The lock order is Target tombstone, Scan, then Task so it serializes with
// Target deletion and result materialization without touching retained history.
func (r *ScanRepository) CancelNextActiveForDeletedTarget(
	ctx context.Context,
	targetID int,
	now time.Time,
) (*TargetCleanupScanCancellation, error) {
	if r == nil || r.db == nil || targetID <= 0 {
		return nil, nil
	}
	now = now.UTC()
	diagnosticsUpdate, err := unavailableEngineExecutionDiagnosticsForSavedPlan(r.db.Dialector.Name())
	if err != nil {
		return nil, err
	}
	var cancellation *TargetCleanupScanCancellation
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := applyTargetCleanupCancellationTimeouts(tx); err != nil {
			return err
		}
		var target struct {
			ID int `gorm:"column:id"`
		}
		if err := tx.Table("target").Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id").
			Where("id = ? AND deleted_at IS NOT NULL", targetID).
			Take(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}

		var scan targetCleanupScanRow
		if err := tx.Table("scan").Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id").
			Where("target_id = ? AND status IN ?", targetID, []string{scanStatusPending, scanStatusRunning}).
			Order("id ASC").
			Take(&scan).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}

		var tasks []targetCleanupTaskRow
		if err := tx.Table("scan_task").Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id, status, assigned_agent_id, assigned_session_id, assigned_session_epoch, assigned_request_id").
			Where("scan_id = ? AND status IN ?", scan.ID, []string{taskStatusBlocked, taskStatusPending, taskStatusRunning}).
			Order("id ASC").
			Find(&tasks).Error; err != nil {
			return err
		}

		cancellation = &TargetCleanupScanCancellation{
			ScanID:             scan.ID,
			CancelledTaskCount: len(tasks),
		}
		for _, task := range tasks {
			if task.Status == taskStatusRunning && taskHasCompleteAssignment(task) {
				cancellation.NotificationCandidates = append(cancellation.NotificationCandidates, TargetCleanupTaskCancelCandidate{
					TaskID: task.ID, AgentID: *task.AssignedAgentID,
				})
			}
		}

		if len(tasks) > 0 {
			if err := tx.Table("scan_task").
				Where("scan_id = ? AND status IN ?", scan.ID, []string{taskStatusBlocked, taskStatusPending, taskStatusRunning}).
				Updates(map[string]any{
					"status":             taskStatusCancelled,
					"completed_at":       now,
					"engine_diagnostics": diagnosticsUpdate,
				}).Error; err != nil {
				return err
			}
		}
		result := tx.Table("scan").
			Where("id = ? AND status IN ?", scan.ID, []string{scanStatusPending, scanStatusRunning}).
			Updates(map[string]any{"status": scanStatusCancelled, "stopped_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			cancellation = nil
		}
		return nil
	})
	return cancellation, err
}

func taskHasCompleteAssignment(task targetCleanupTaskRow) bool {
	return task.AssignedAgentID != nil && *task.AssignedAgentID > 0 &&
		task.AssignedSessionID != nil && strings.TrimSpace(*task.AssignedSessionID) != "" &&
		task.AssignedSessionEpoch != nil && *task.AssignedSessionEpoch > 0 &&
		task.AssignedRequestID != nil && strings.TrimSpace(*task.AssignedRequestID) != ""
}

func applyTargetCleanupCancellationTimeouts(tx *gorm.DB) error {
	if tx == nil || tx.Dialector.Name() != "postgres" {
		return nil
	}
	if err := tx.Exec("SELECT set_config('lock_timeout', '5s', true)").Error; err != nil {
		return err
	}
	return tx.Exec("SELECT set_config('statement_timeout', '30s', true)").Error
}
