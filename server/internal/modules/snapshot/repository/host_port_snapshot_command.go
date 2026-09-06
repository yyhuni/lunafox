package repository

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm/clause"
)

// BatchCreate creates multiple host-port snapshots, ignoring duplicates
func (r *HostPortSnapshotRepository) BatchCreate(snapshots []snapshotdomain.HostPortSnapshot) (int64, error) {
	return r.BatchCreateContext(context.Background(), snapshots)
}

// BatchCreateContext persists host-port evidence under the caller deadline.
func (r *HostPortSnapshotRepository) BatchCreateContext(ctx context.Context, snapshots []snapshotdomain.HostPortSnapshot) (int64, error) {
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

		// Keep writes on the ResultIngest transaction so Scan/Target locks are not
		// re-acquired by a second connection during the same materialization.
		result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
		if result.Error != nil {
			return totalAffected, result.Error
		}
		totalAffected += result.RowsAffected
	}

	return totalAffected, nil
}
