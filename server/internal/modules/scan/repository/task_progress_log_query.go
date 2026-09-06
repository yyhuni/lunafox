package repository

import (
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
)

// FindByScanIDWithCursor finds task progress logs by scan ID with cursor pagination.
func (r *TaskProgressLogRepository) FindByScanIDWithCursor(scanID int, afterID int64, limit int) ([]TaskProgressLogRecord, error) {
	var logs []model.TaskProgressLog

	query := r.db.Model(&model.TaskProgressLog{}).
		Where("task_progress_log.scan_id = ?", scanID)
	if afterID > 0 {
		query = query.Where("task_progress_log.id > ?", afterID)
	}

	err := query.Order("task_progress_log.id ASC").Limit(limit).Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return taskProgressLogModelListToRecord(logs), nil
}
