package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UpgradeCancellationReason is a stable, user-visible classification. It is
// intentionally independent of an error string so retries and audit readers
// can recognize maintenance cancellations consistently.
const UpgradeCancellationReason = "system_upgrade_maintenance"

// UpgradeScanStopOutcome contains only state committed by the maintenance
// transaction. Running-task assignments are returned for post-commit,
// best-effort task_cancel delivery.
type UpgradeScanStopOutcome struct {
	CancelledScanCount     int
	CancelledTaskCount     int
	NotificationCandidates []BatchScanStopNotificationCandidate
}

// StopAllActiveScansForUpgrade atomically cancels every non-terminal Scan and
// its active Tasks. The Scan rows are locked before Task rows, matching the
// normal stop lifecycle and giving result-ingest a deterministic cancellation
// fence. No pagination or natural task drain occurs.
func (r *ScanRepository) StopAllActiveScansForUpgrade(ctx context.Context, operationID string, stoppedAt time.Time) (*UpgradeScanStopOutcome, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrRecordNotFound
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	operationID = strings.TrimSpace(operationID)
	if operationID == "" || len(operationID) > 128 {
		return nil, fmt.Errorf("upgrade operation id is invalid")
	}
	stoppedAt = stoppedAt.UTC()
	if stoppedAt.IsZero() {
		stoppedAt = time.Now().UTC()
	}
	diagnosticsUpdate, err := unavailableEngineExecutionDiagnosticsForSavedPlan(r.db.Name())
	if err != nil {
		return nil, err
	}
	outcome := &UpgradeScanStopOutcome{}
	err = dbtx.Resolve(ctx, r.db).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var scans []scanStopRow
		query := tx.WithContext(ctx).Table("scan").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "status").
			Where("status IN ? AND deleted_at IS NULL", []string{scanStatusPending, scanStatusRunning}).
			Order("id ASC")
		if err := query.Find(&scans).Error; err != nil {
			return err
		}
		if len(scans) == 0 {
			return nil
		}
		activeIDs := make([]int, 0, len(scans))
		for _, scan := range scans {
			activeIDs = append(activeIDs, scan.ID)
		}

		var tasks []scanStopTaskRow
		if err := tx.WithContext(ctx).Table("scan_task").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "scan_id", "status", "assigned_agent_id", "assigned_session_id", "assigned_session_epoch", "assigned_request_id").
			Where("scan_id IN ? AND status IN ?", activeIDs, []string{taskStatusBlocked, taskStatusPending, taskStatusRunning}).
			Order("scan_id ASC, id ASC").Find(&tasks).Error; err != nil {
			return err
		}
		outcome.CancelledScanCount = len(activeIDs)
		outcome.CancelledTaskCount = len(tasks)
		for _, task := range tasks {
			if task.Status == taskStatusRunning && scanStopTaskHasCompleteAssignment(task) {
				outcome.NotificationCandidates = append(outcome.NotificationCandidates, BatchScanStopNotificationCandidate{
					ScanID: task.ScanID, TaskID: task.ID, AgentID: *task.AssignedAgentID,
				})
			}
		}

		if len(tasks) > 0 {
			// These audit columns are intentionally read-only on the projection
			// model used by ordinary Scan reads. Use the table update explicitly so
			// the maintenance transaction still persists the cancellation fence.
			if err := tx.WithContext(ctx).Table("scan_task").
				Where("id IN ? AND status IN ?", scanStopTaskIDs(tasks), []string{taskStatusBlocked, taskStatusPending, taskStatusRunning}).
				Updates(map[string]any{
					"status":                    taskStatusCancelled,
					"completed_at":              stoppedAt,
					"engine_diagnostics":        diagnosticsUpdate,
					"cancellation_reason":       UpgradeCancellationReason,
					"cancellation_operation_id": operationID,
				}).Error; err != nil {
				return err
			}
		}
		result := tx.WithContext(ctx).Table("scan").
			Where("id IN ? AND status IN ?", activeIDs, []string{scanStatusPending, scanStatusRunning}).
			Updates(map[string]any{
				"status":                    scanStatusCancelled,
				"stopped_at":                stoppedAt,
				"cancellation_reason":       UpgradeCancellationReason,
				"cancellation_operation_id": operationID,
			})
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

var _ interface {
	StopAllActiveScansForUpgrade(context.Context, string, time.Time) (*UpgradeScanStopOutcome, error)
} = (*ScanRepository)(nil)
