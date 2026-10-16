package repository

import (
	"context"
	"sort"

	"gorm.io/gorm"

	"gorm.io/gorm/clause"
)

func (r *TaskProgressLogRepository) BatchCreateTaskProgressLogs(ctx context.Context, logs []TaskProgressLogRecord) (int, int, error) {
	if len(logs) == 0 {
		return 0, 0, nil
	}
	accepted := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Name() == "postgres" {
			// Sequence allocation alone does not order commits. Serialize each
			// Scan before allocating IDs so an afterID reader cannot skip a
			// lower-ID row committed after a concurrent higher-ID writer.
			scanIDs := make([]int, 0, len(logs))
			seen := make(map[int]struct{})
			for _, entry := range logs {
				if _, exists := seen[entry.ScanID]; !exists {
					scanIDs = append(scanIDs, entry.ScanID)
					seen[entry.ScanID] = struct{}{}
				}
			}
			sort.Ints(scanIDs)
			for _, scanID := range scanIDs {
				// A dedicated namespace avoids coupling log traffic to Scan
				// lifecycle row locks; sorted IDs avoid multi-Scan deadlocks.
				if err := tx.Exec("SELECT pg_advisory_xact_lock(?, ?)", taskProgressLogAdvisoryLockNamespace, scanID).Error; err != nil {
					return err
				}
			}
		}
		modelLogs := taskProgressLogRecordListToModel(logs)
		// Agents may retry the same request, so serialization must retain the
		// existing task/request/sequence identity at the database boundary.
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "scan_id"}, {Name: "task_id"}, {Name: "request_id"}, {Name: "sequence"}},
			DoNothing: true,
		}).CreateInBatches(modelLogs, 100)
		if result.Error != nil {
			return result.Error
		}
		accepted = int(result.RowsAffected)
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	duplicates := len(logs) - accepted
	if duplicates < 0 {
		duplicates = 0
	}
	return accepted, duplicates, nil
}

const taskProgressLogAdvisoryLockNamespace = 0x4c464c47
