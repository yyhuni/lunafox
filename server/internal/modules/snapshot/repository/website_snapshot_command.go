package repository

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm/clause"
)

// BatchCreate creates multiple website snapshots, ignoring duplicates
func (r *WebsiteSnapshotRepository) BatchCreate(snapshots []snapshotdomain.WebsiteSnapshot) (int64, error) {
	return r.BatchCreateContext(context.Background(), snapshots)
}

// BatchCreateContext persists website evidence under the caller deadline.
func (r *WebsiteSnapshotRepository) BatchCreateContext(ctx context.Context, snapshots []snapshotdomain.WebsiteSnapshot) (int64, error) {
	if len(snapshots) == 0 {
		return 0, nil
	}

	modelSnapshots := websiteSnapshotDomainListToModel(snapshots)
	var totalAffected int64

	batchSize := 500
	for i := 0; i < len(modelSnapshots); i += batchSize {
		end := i + batchSize
		if end > len(modelSnapshots) {
			end = len(modelSnapshots)
		}
		batch := modelSnapshots[i:end]

		result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "scan_id"}, {Name: "url"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"host", "title", "status_code", "content_length", "location", "webserver",
				"content_type", "tech", "response_body", "vhost", "response_headers",
			}),
		}).Create(&batch)
		if result.Error != nil {
			return totalAffected, result.Error
		}
		totalAffected += result.RowsAffected
	}

	return totalAffected, nil
}
