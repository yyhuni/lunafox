package repository

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BatchDelete deletes multiple screenshots by IDs
func (r *ScreenshotRepository) BatchDelete(ids []int) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.Where("id IN ?", ids).Delete(&model.Screenshot{})
	return result.RowsAffected, result.Error
}

// BatchUpsert creates or updates multiple screenshots
func (r *ScreenshotRepository) BatchUpsert(screenshots []assetdomain.Screenshot) (int64, error) {
	return r.BatchUpsertContext(context.Background(), screenshots)
}

func (r *ScreenshotRepository) BatchUpsertContext(ctx context.Context, screenshots []assetdomain.Screenshot) (int64, error) {
	if len(screenshots) == 0 {
		return 0, nil
	}

	modelScreenshots := screenshotDomainListToModel(screenshots)
	var totalAffected int64

	batchSize := 500
	for i := 0; i < len(modelScreenshots); i += batchSize {
		end := i + batchSize
		if end > len(modelScreenshots) {
			end = len(modelScreenshots)
		}
		batch := modelScreenshots[i:end]

		result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "target_id"}, {Name: "url"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"status_code": gorm.Expr("EXCLUDED.status_code"),
				"image":       gorm.Expr("EXCLUDED.image"),
				"updated_at":  gorm.Expr("CURRENT_TIMESTAMP"),
			}),
		}).Create(&batch)
		if result.Error != nil {
			return totalAffected, result.Error
		}
		totalAffected += result.RowsAffected
	}

	return totalAffected, nil
}
