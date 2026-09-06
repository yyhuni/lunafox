package repository

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm/clause"
)

// BatchCreate creates multiple subdomain snapshots, ignoring duplicates
func (r *SubdomainSnapshotRepository) BatchCreate(snapshots []snapshotdomain.SubdomainSnapshot) (int64, error) {
	return r.BatchCreateContext(context.Background(), snapshots)
}

// BatchCreateContext persists subdomain evidence under the caller deadline.
func (r *SubdomainSnapshotRepository) BatchCreateContext(ctx context.Context, snapshots []snapshotdomain.SubdomainSnapshot) (int64, error) {
	if len(snapshots) == 0 {
		return 0, nil
	}

	modelSnapshots := subdomainSnapshotDomainListToModel(snapshots)
	var totalAffected int64

	batchSize := 500
	for i := 0; i < len(modelSnapshots); i += batchSize {
		end := i + batchSize
		if end > len(modelSnapshots) {
			end = len(modelSnapshots)
		}
		batch := modelSnapshots[i:end]

		// Result ingest keeps Scan/Target locks in the ambient transaction. A root
		// connection here would wait on those locks and self-deadlock the batch.
		result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
		if result.Error != nil {
			return totalAffected, result.Error
		}
		totalAffected += result.RowsAffected
	}

	return totalAffected, nil
}
