package repository

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm/clause"
)

// BulkCreate creates multiple host-port snapshots, ignoring duplicates
func (r *HostPortSnapshotRepository) BulkCreate(snapshots []snapshotdomain.HostPortSnapshot) (int64, error) {
	if len(snapshots) == 0 {
		return 0, nil
	}

	modelSnapshots := hostPortSnapshotDomainListToModel(snapshots)
	var totalAffected int64

	batchSize := 500
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
