package repository

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm/clause"
)

// BatchCreate creates multiple directory snapshots, ignoring duplicates
func (r *DirectorySnapshotRepository) BatchCreate(snapshots []snapshotdomain.DirectorySnapshot) (int64, error) {
	return r.BatchCreateContext(context.Background(), snapshots)
}

func (r *DirectorySnapshotRepository) BatchCreateContext(ctx context.Context, snapshots []snapshotdomain.DirectorySnapshot) (int64, error) {
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

		result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "scan_id"}, {Name: "url"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "content_length", "content_type", "duration"}),
		}).Create(&batch)
		if result.Error != nil {
			return totalAffected, result.Error
		}
		totalAffected += result.RowsAffected
	}

	return totalAffected, nil
}
