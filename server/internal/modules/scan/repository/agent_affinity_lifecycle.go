package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const agentDeletedFailureMessage = "Selected Agent was deleted before task execution"

// DeleteAgent serializes physical Agent removal with pinned-task claiming.
// The Agent row lock is also acquired by a pinned claim before it transitions
// a task to running, so no deleted Agent can win a pending-task claim.
func (r *ScanRepository) DeleteAgent(ctx context.Context, agentID int) error {
	if r == nil || r.db == nil || ctx == nil || agentID <= 0 {
		return fmt.Errorf("agent deletion scope is invalid")
	}
	diagnosticsUpdate, err := unavailableEngineExecutionDiagnosticsForSavedPlan(r.db.Dialector.Name())
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Table("agent").Select("id").Where("id = ?", agentID)
		if tx.Dialector.Name() == "postgres" {
			query = query.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var row struct{ ID int }
		if err := query.Take(&row).Error; err != nil {
			return err
		}

		now := time.Now().UTC()
		failedTasks := tx.Exec(`
			UPDATE scan_task
			SET status = ?, completed_at = ?, terminal_reconciliation_pending = FALSE,
				error_message = ?, failure_kind = ?, failure_detail = '',
				engine_diagnostics = ?
			WHERE status IN (?, ?)
				AND scan_id IN (
					SELECT id FROM scan
					WHERE agent_id = ? AND assignment_mode = 'pinned' AND deleted_at IS NULL
				)`, taskStatusFailed, now, agentDeletedFailureMessage, "agent_deleted", diagnosticsUpdate, taskStatusPending, taskStatusBlocked, agentID)
		if failedTasks.Error != nil {
			return failedTasks.Error
		}

		// Only scans without work already executing can converge here. Running
		// work remains owned by its session-fence disconnect path.
		converged := tx.Exec(`
			UPDATE scan AS s
			SET status = ?, stopped_at = ?, error_message = ?, failure_kind = ?
			WHERE s.agent_id = ? AND s.assignment_mode = 'pinned' AND s.deleted_at IS NULL
				AND s.status IN (?, ?)
				AND EXISTS (
					SELECT 1 FROM scan_task AS st
					WHERE st.scan_id = s.id AND st.failure_kind = 'agent_deleted'
				)
				AND NOT EXISTS (
					SELECT 1 FROM scan_task AS active
					WHERE active.scan_id = s.id AND active.status IN (?, ?, ?)
				)`, scanStatusFailed, now, agentDeletedFailureMessage, "agent_deleted", agentID, scanStatusPending, scanStatusRunning, taskStatusPending, taskStatusBlocked, taskStatusRunning)
		if converged.Error != nil {
			return converged.Error
		}

		result := tx.Exec("DELETE FROM agent WHERE id = ?", agentID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func requireClaimingAgentExists(tx *gorm.DB, agentID int) error {
	if tx == nil || agentID <= 0 {
		return errors.New("claiming Agent is required")
	}
	query := tx.Table("agent").Select("id").Where("id = ?", agentID)
	if tx.Dialector.Name() == "postgres" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row struct{ ID int }
	return query.Take(&row).Error
}
