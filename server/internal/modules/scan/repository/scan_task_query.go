package repository

import (
	"context"
	"fmt"

	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
)

// GetByID finds a scan task by ID.
func (r *scanTaskRepository) GetByID(ctx context.Context, id int) (*ScanTaskRecord, error) {
	var task model.ScanTask
	err := r.db.WithContext(ctx).First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return scanTaskModelToRecord(&task), nil
}

// PullTask atomically pulls a pending task and assigns it to an agent.
func (r *scanTaskRepository) PullTask(ctx context.Context, agentID int) (*ScanTaskRecord, error) {
	var task model.ScanTask
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("scan task repository is not initialized")
	}
	err := r.db.WithContext(ctx).Raw(
		pullTaskSQL,
		agentID,
	).Scan(&task).Error
	if err != nil {
		return nil, err
	}
	if task.ID == 0 {
		return nil, nil
	}
	return scanTaskModelToRecord(&task), nil
}

// ListFailedByScanID returns all failed tasks for a scan.
func (r *scanTaskRepository) ListFailedByScanID(ctx context.Context, scanID int) ([]ScanTaskRecord, error) {
	var tasks []model.ScanTask
	err := r.db.WithContext(ctx).
		Where("scan_id = ? AND status = ?", scanID, taskStatusFailed).
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	results := make([]ScanTaskRecord, 0, len(tasks))
	for index := range tasks {
		results = append(results, *scanTaskModelToRecord(&tasks[index]))
	}
	return results, nil
}

// GetStatusCountsByScanID returns task status counts for a scan.
func (r *scanTaskRepository) GetStatusCountsByScanID(ctx context.Context, scanID int) (pending, running, completed, failed, cancelled int, err error) {
	var results []struct {
		Status string
		Count  int
	}
	err = r.db.WithContext(ctx).
		Model(&model.ScanTask{}).
		Select("status, COUNT(*) as count").
		Where("scan_id = ?", scanID).
		Group("status").
		Scan(&results).Error
	if err != nil {
		return
	}
	for _, row := range results {
		switch row.Status {
		case taskStatusPending:
			pending = row.Count
		case taskStatusBlocked:
			pending += row.Count
		case taskStatusRunning:
			running = row.Count
		case taskStatusCompleted:
			completed = row.Count
		case taskStatusFailed:
			failed = row.Count
		case taskStatusCancelled:
			cancelled = row.Count
		}
	}
	return
}

// CountActiveByScanAndStage returns the count of pending/running tasks for a stage.
func (r *scanTaskRepository) CountActiveByScanAndStage(ctx context.Context, scanID, stage int) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&struct{}{}).
		Table("scan_task").
		Where("scan_id = ? AND stage = ? AND status IN ?", scanID, stage, []string{taskStatusPending, taskStatusRunning}).
		Count(&count).Error
	return int(count), err
}
