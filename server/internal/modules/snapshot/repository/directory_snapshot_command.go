package repository

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm/clause"
)

// BulkCreate creates multiple directory snapshots, ignoring duplicates
func (r *DirectorySnapshotRepository) BulkCreate(snapshots []snapshotdomain.DirectorySnapshot) (int64, error) {
	if len(snapshots) == 0 {
		return 0, nil
	}

	modelSnapshots := directorySnapshotDomainListToModel(snapshots)
	var totalAffected int64

	batchSize := 100
	for i := 0; i < len(modelSnapshots); i += batchSize {
		end := i + batchSize
		if end > len(modelSnapshots) {
			end = len(modelSnapshots)
		}
		batch := modelSnapshots[i:end]

		result := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
		if result.Error != nil {
			return totalAffected, result.Error
		}
		totalAffected += result.RowsAffected
	}

	return totalAffected, nil
}
