package repository

// BulkCreate creates multiple scan logs
func (r *ScanLogRepository) BulkCreate(logs []ScanLogRecord) error {
	if len(logs) == 0 {
		return nil
	}
	modelLogs := scanLogRecordListToModel(logs)
	return r.db.CreateInBatches(modelLogs, 100).Error
}
