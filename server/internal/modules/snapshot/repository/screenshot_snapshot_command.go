package repository

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BulkUpsert creates or updates multiple screenshot snapshots.
func (r *ScreenshotSnapshotRepository) BulkUpsert(snapshots []snapshotdomain.ScreenshotSnapshot) (int64, error) {
	if len(snapshots) == 0 {
		return 0, nil
	}

	modelSnapshots := screenshotSnapshotDomainListToModel(snapshots)
	var totalAffected int64

	batchSize := 500
	for i := 0; i < len(modelSnapshots); i += batchSize {
		end := i + batchSize
		if end > len(modelSnapshots) {
			end = len(modelSnapshots)
		}
		batch := modelSnapshots[i:end]

		result := r.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "scan_id"}, {Name: "url"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"status_code": gorm.Expr("COALESCE(EXCLUDED.status_code, screenshot_snapshot.status_code)"),
				"image":       gorm.Expr("COALESCE(EXCLUDED.image, screenshot_snapshot.image)"),
			}),
		}).Create(&batch)
		if result.Error != nil {
			return totalAffected, result.Error
		}
		totalAffected += result.RowsAffected
	}

	return totalAffected, nil
}
