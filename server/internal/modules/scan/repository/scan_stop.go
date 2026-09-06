package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ScanStopNotificationCandidate is a running Task with a complete durable
// assignment that may receive a best-effort cancellation after commit.
type ScanStopNotificationCandidate struct {
	TaskID  int
	AgentID int
}

// ScanStopOutcome contains only state committed by StopActiveScan.
type ScanStopOutcome struct {
	ScanID                 int
	CancelledTaskCount     int
	NotificationCandidates []ScanStopNotificationCandidate
}

// BatchScanStopNotificationCandidate retains the owning Scan identity for a
// post-commit notification emitted from a multi-Scan transaction.
type BatchScanStopNotificationCandidate struct {
	ScanID  int
	TaskID  int
	AgentID int
}

// BatchScanStopOutcome contains only state committed by BatchStopActiveScans.
type BatchScanStopOutcome struct {
	StoppedCount           int
	SkippedCount           int
	RevokedTaskCount       int
	NotificationCandidates []BatchScanStopNotificationCandidate
}

type scanStopRow struct {
	ID     int    `gorm:"column:id"`
	Status string `gorm:"column:status"`
}

type scanStopTaskRow struct {
	ID                   int     `gorm:"column:id"`
	ScanID               int     `gorm:"column:scan_id"`
	Status               string  `gorm:"column:status"`
	AssignedAgentID      *int    `gorm:"column:assigned_agent_id"`
	AssignedSessionID    *string `gorm:"column:assigned_session_id"`
	AssignedSessionEpoch *int64  `gorm:"column:assigned_session_epoch"`
	AssignedRequestID    *string `gorm:"column:assigned_request_id"`
}

// BatchStopActiveScans stops a bounded set of Scans in one transaction. It
// locks every Scan before any Task so concurrent batch stops and result ingest
// observe one deterministic lifecycle order; terminal rows are a committed
// skipped result rather than a race-induced failure.
func (r *ScanRepository) BatchStopActiveScans(ctx context.Context, scanIDs []int, stoppedAt time.Time) (*BatchScanStopOutcome, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrRecordNotFound
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	orderedIDs, err := normalizeBatchScanStopIDs(scanIDs)
	if err != nil {
		return nil, err
	}
	stoppedAt = stoppedAt.UTC()
	diagnosticsUpdate, err := unavailableEngineExecutionDiagnosticsForSavedPlan(r.db.Dialector.Name())
	if err != nil {
		return nil, err
	}

	var outcome *BatchScanStopOutcome
	err = dbtx.Resolve(ctx, r.db).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var scans []scanStopRow
		if err := tx.WithContext(ctx).Table("scan").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "status").
			Where("id IN ? AND deleted_at IS NULL", orderedIDs).
			Order("id ASC").
			Find(&scans).Error; err != nil {
			return err
		}
		if len(scans) != len(orderedIDs) {
			return gorm.ErrRecordNotFound
		}

		activeIDs := make([]int, 0, len(scans))
		for _, scan := range scans {
			if scan.Status == scanStatusPending || scan.Status == scanStatusRunning {
				activeIDs = append(activeIDs, scan.ID)
			}
		}
		outcome = &BatchScanStopOutcome{
			StoppedCount: len(activeIDs),
			SkippedCount: len(scans) - len(activeIDs),
		}
		if len(activeIDs) == 0 {
			return nil
		}

		var tasks []scanStopTaskRow
		if err := tx.WithContext(ctx).Table("scan_task").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "scan_id", "status", "assigned_agent_id", "assigned_session_id", "assigned_session_epoch", "assigned_request_id").
			Where("scan_id IN ? AND status IN ?", activeIDs, []string{taskStatusBlocked, taskStatusPending, taskStatusRunning}).
			Order("scan_id ASC, id ASC").
			Find(&tasks).Error; err != nil {
			return err
		}

		outcome.RevokedTaskCount = len(tasks)
		for _, task := range tasks {
			if task.Status == taskStatusRunning && scanStopTaskHasCompleteAssignment(task) {
				outcome.NotificationCandidates = append(outcome.NotificationCandidates, BatchScanStopNotificationCandidate{
					ScanID:  task.ScanID,
					TaskID:  task.ID,
					AgentID: *task.AssignedAgentID,
				})
			}
		}

		if len(tasks) > 0 {
			if err := tx.WithContext(ctx).Model(&model.ScanTask{}).
				Where("id IN ?", scanStopTaskIDs(tasks)).
				Updates(map[string]any{
					"status":             taskStatusCancelled,
					"completed_at":       stoppedAt,
					"engine_diagnostics": diagnosticsUpdate,
				}).Error; err != nil {
				return err
			}
		}
		result := tx.WithContext(ctx).Model(&model.Scan{}).
			Where("id IN ? AND status IN ?", activeIDs, []string{scanStatusPending, scanStatusRunning}).
			Updates(map[string]any{"status": scanStatusCancelled, "stopped_at": stoppedAt})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != int64(len(activeIDs)) {
			return scandomain.ErrScanCannotStop
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return outcome, nil
}

func normalizeBatchScanStopIDs(scanIDs []int) ([]int, error) {
	if len(scanIDs) == 0 || len(scanIDs) > 100 {
		return nil, fmt.Errorf("batch scan stop requires between 1 and 100 scans")
	}
	orderedIDs := append([]int(nil), scanIDs...)
	sort.Ints(orderedIDs)
	for index, scanID := range orderedIDs {
		if scanID <= 0 {
			return nil, fmt.Errorf("batch scan stop requires positive scan IDs")
		}
		if index > 0 && scanID == orderedIDs[index-1] {
			return nil, fmt.Errorf("batch scan stop requires unique scan IDs")
		}
	}
	return orderedIDs, nil
}

// StopActiveScan atomically cancels one active Scan and all of its active
// Tasks. It locks Scan before Tasks so a concurrent result-ingest fence sees a
// single durable lifecycle transition rather than a partially cancelled batch.
func (r *ScanRepository) StopActiveScan(ctx context.Context, scanID int, stoppedAt time.Time) (*ScanStopOutcome, error) {
	if r == nil || r.db == nil || scanID <= 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	stoppedAt = stoppedAt.UTC()
	diagnosticsUpdate, err := unavailableEngineExecutionDiagnosticsForSavedPlan(r.db.Dialector.Name())
	if err != nil {
		return nil, err
	}
	var outcome *ScanStopOutcome
	err = dbtx.Resolve(ctx, r.db).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var scan scanStopRow
		if err := tx.WithContext(ctx).Table("scan").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "status").
			Where("id = ? AND deleted_at IS NULL", scanID).
			Take(&scan).Error; err != nil {
			return err
		}
		if scan.Status != scanStatusPending && scan.Status != scanStatusRunning {
			return scandomain.ErrScanCannotStop
		}

		var tasks []scanStopTaskRow
		if err := tx.WithContext(ctx).Table("scan_task").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "status", "assigned_agent_id", "assigned_session_id", "assigned_session_epoch", "assigned_request_id").
			Where("scan_id = ? AND status IN ?", scanID, []string{taskStatusBlocked, taskStatusPending, taskStatusRunning}).
			Order("id ASC").
			Find(&tasks).Error; err != nil {
			return err
		}

		outcome = &ScanStopOutcome{ScanID: scan.ID, CancelledTaskCount: len(tasks)}
		for _, task := range tasks {
			if task.Status == taskStatusRunning && scanStopTaskHasCompleteAssignment(task) {
				outcome.NotificationCandidates = append(outcome.NotificationCandidates, ScanStopNotificationCandidate{
					TaskID:  task.ID,
					AgentID: *task.AssignedAgentID,
				})
			}
		}

		if len(tasks) > 0 {
			if err := tx.WithContext(ctx).Model(&model.ScanTask{}).
				Where("id IN ?", scanStopTaskIDs(tasks)).
				Updates(map[string]any{
					"status":             taskStatusCancelled,
					"completed_at":       stoppedAt,
					"engine_diagnostics": diagnosticsUpdate,
				}).Error; err != nil {
				return err
			}
		}
		result := tx.WithContext(ctx).Model(&model.Scan{}).
			Where("id = ? AND status IN ?", scan.ID, []string{scanStatusPending, scanStatusRunning}).
			Updates(map[string]any{"status": scanStatusCancelled, "stopped_at": stoppedAt})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return scandomain.ErrScanCannotStop
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return outcome, nil
}

func scanStopTaskHasCompleteAssignment(task scanStopTaskRow) bool {
	return task.AssignedAgentID != nil && *task.AssignedAgentID > 0 &&
		task.AssignedSessionID != nil && strings.TrimSpace(*task.AssignedSessionID) != "" &&
		task.AssignedSessionEpoch != nil && *task.AssignedSessionEpoch > 0 &&
		task.AssignedRequestID != nil && strings.TrimSpace(*task.AssignedRequestID) != ""
}

func scanStopTaskIDs(tasks []scanStopTaskRow) []int {
	ids := make([]int, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	return ids
}
