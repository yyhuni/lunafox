package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
)

// UpdateStatus updates task status.
func (r *scanTaskRepository) UpdateStatus(
	ctx context.Context,
	id int,
	status string,
	failure *scandomain.FailureDetail,
) error {
	var err error
	failure, err = normalizeTaskFailureDetail(status, failure)
	if err != nil {
		return err
	}
	updates := map[string]interface{}{
		"status": status,
	}
	if status == taskStatusCompleted || status == taskStatusFailed || status == taskStatusCancelled {
		now := time.Now().UTC()
		updates["completed_at"] = now
	}
	if failure != nil {
		updates["error_message"] = failure.Message
		updates["failure_kind"] = failure.Kind
	} else {
		updates["error_message"] = ""
		updates["failure_kind"] = ""
	}

	return r.db.WithContext(ctx).
		Model(&model.ScanTask{}).
		Where("id = ?", id).
		Updates(updates).
		Error
}

// FailPulledTask marks a pulled task as failed for non-recoverable scheduler errors.
func (r *scanTaskRepository) FailPulledTask(
	ctx context.Context,
	id int,
	failure *scandomain.FailureDetail,
) error {
	var err error
	failure, err = normalizeFailedFailureDetail(failure)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":        taskStatusFailed,
		"agent_id":      nil,
		"started_at":    nil,
		"completed_at":  &now,
		"error_message": failure.Message,
		"failure_kind":  failure.Kind,
	}

	return r.db.WithContext(ctx).
		Model(&model.ScanTask{}).
		Where("id = ? AND status = ?", id, taskStatusRunning).
		Updates(updates).
		Error
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

func normalizeTaskFailureDetail(status string, failure *scandomain.FailureDetail) (*scandomain.FailureDetail, error) {
	if strings.TrimSpace(status) != taskStatusFailed {
		return nil, nil
	}
	return normalizeFailedFailureDetail(failure)
}

func normalizeFailedFailureDetail(failure *scandomain.FailureDetail) (*scandomain.FailureDetail, error) {
	if failure == nil {
		return nil, fmt.Errorf("failed task requires failure detail")
	}
	message := strings.TrimSpace(failure.Message)
	if message == "" {
		return nil, fmt.Errorf("failed task requires non-empty failure message")
	}
	kind := canonicalTaskFailureKind(strings.TrimSpace(failure.Kind))
	if kind == "" {
		kind = "unknown"
	}
	return &scandomain.FailureDetail{Kind: kind, Message: message}, nil
}

func canonicalTaskFailureKind(value string) string {
	return strings.TrimSpace(value)
}

// UnlockNextStage promotes the next blocked stage to pending.
func (r *scanTaskRepository) UnlockNextStage(ctx context.Context, scanID, stage int) (int64, error) {
	result := r.db.WithContext(ctx).Exec(unlockNextStageSQL, scanID, stage, scanID)
	return result.RowsAffected, result.Error
}
