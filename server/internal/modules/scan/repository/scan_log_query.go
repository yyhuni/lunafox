package repository

import (
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
)

// FindByScanIDWithCursor finds logs by scan ID with cursor pagination
func (r *ScanLogRepository) FindByScanIDWithCursor(scanID int, afterID int64, limit int) ([]ScanLogRecord, error) {
	var logs []model.ScanLog

	query := r.db.Where("scan_id = ?", scanID)
	if afterID > 0 {
		query = query.Where("id > ?", afterID)
	}

	err := query.Order("id ASC").Limit(limit).Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return scanLogModelListToRecord(logs), nil
}
