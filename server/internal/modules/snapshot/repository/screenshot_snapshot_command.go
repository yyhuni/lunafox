package repository

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm/clause"
)

// BatchUpsert creates or updates multiple screenshot snapshots.
func (r *ScreenshotSnapshotRepository) BatchUpsert(snapshots []snapshotdomain.ScreenshotSnapshot) (int64, error) {
	return r.BatchUpsertContext(context.Background(), snapshots)
}

func (r *ScreenshotSnapshotRepository) BatchUpsertContext(ctx context.Context, snapshots []snapshotdomain.ScreenshotSnapshot) (int64, error) {
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

		result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "scan_id"}, {Name: "url"}},
			DoUpdates: clause.AssignmentColumns([]string{"status_code", "image"}),
		}).Create(&batch)
		if result.Error != nil {
			return totalAffected, result.Error
		}
		totalAffected += result.RowsAffected
	}

	return totalAffected, nil
}
