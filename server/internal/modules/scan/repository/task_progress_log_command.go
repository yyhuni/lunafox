package repository

import (
	"context"

	"gorm.io/gorm/clause"
)

func (r *TaskProgressLogRepository) BatchCreateTaskProgressLogs(ctx context.Context, logs []TaskProgressLogRecord) (int, int, error) {
	if len(logs) == 0 {
		return 0, 0, nil
	}
	modelLogs := taskProgressLogRecordListToModel(logs)
	// Agents may retry the same progress-log request, so the runtime identity
	// must stay idempotent at the database boundary.
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "scan_id"}, {Name: "task_id"}, {Name: "request_id"}, {Name: "sequence"}},
		DoNothing: true,
	}).CreateInBatches(modelLogs, 100)
	if result.Error != nil {
		return 0, 0, result.Error
	}
	accepted := int(result.RowsAffected)
	duplicates := len(logs) - accepted
	if duplicates < 0 {
		duplicates = 0
	}
	return accepted, duplicates, nil
}
