package repository

import (
	"context"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
)

// UpdateStatus updates task status.
func (r *scanTaskRepository) UpdateStatus(ctx context.Context, id int, status string, errorMessage string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if status == taskStatusCompleted || status == taskStatusFailed || status == taskStatusCancelled {
		now := time.Now().UTC()
		updates["completed_at"] = now
	}
	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}
	return r.db.WithContext(ctx).Model(&model.ScanTask{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// CancelTasksByScanID cancels pending/running tasks for a scan and returns affected task/agent pairs.
func (r *scanTaskRepository) CancelTasksByScanID(ctx context.Context, scanID int) ([]CancelledTaskInfo, error) {
	var cancelled []CancelledTaskInfo
	err := r.db.WithContext(ctx).Raw(`
		UPDATE scan_task
		SET status = '`+taskStatusCancelled+`',
		    completed_at = NOW()
		WHERE scan_id = ?
		  AND status IN ('`+taskStatusPending+`', '`+taskStatusRunning+`', '`+taskStatusBlocked+`')
		RETURNING id, agent_id
	`, scanID).Scan(&cancelled).Error
	if err != nil {
		return nil, err
	}
	return cancelled, nil
}

// FailTasksForOfflineAgent marks running tasks as failed for an offline agent.
func (r *scanTaskRepository) FailTasksForOfflineAgent(ctx context.Context, agentID int) error {
	return r.db.WithContext(ctx).Exec(failTasksSQL, agentID).Error
}

// UnlockNextStage promotes the next blocked stage to pending.
func (r *scanTaskRepository) UnlockNextStage(ctx context.Context, scanID, stage int) (int64, error) {
	result := r.db.WithContext(ctx).Exec(unlockNextStageSQL, scanID, stage, scanID)
	return result.RowsAffected, result.Error
}
