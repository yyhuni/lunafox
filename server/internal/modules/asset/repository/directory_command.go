package repository

import (
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BulkCreate creates multiple directories, ignoring duplicates
func (r *DirectoryRepository) BulkCreate(directories []assetdomain.Directory) (int, error) {
	if len(directories) == 0 {
		return 0, nil
	}

	modelDirectories := directoryDomainListToModel(directories)
	var totalAffected int

	batchSize := 500
	for i := 0; i < len(modelDirectories); i += batchSize {
		end := i + batchSize
		if end > len(modelDirectories) {
			end = len(modelDirectories)
		}
		batch := modelDirectories[i:end]

		result := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
		if result.Error != nil {
			return totalAffected, result.Error
		}
		totalAffected += int(result.RowsAffected)
	}

	return totalAffected, nil
}

// BulkDelete deletes multiple directories by IDs
func (r *DirectoryRepository) BulkDelete(ids []int) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.Where("id IN ?", ids).Delete(&model.Directory{})
	return result.RowsAffected, result.Error
}

// BulkUpsert creates or updates multiple directories
func (r *DirectoryRepository) BulkUpsert(directories []assetdomain.Directory) (int64, error) {
	if len(directories) == 0 {
		return 0, nil
	}

	modelDirectories := directoryDomainListToModel(directories)
	var totalAffected int64

	batchSize := 100
	for i := 0; i < len(modelDirectories); i += batchSize {
		end := i + batchSize
		if end > len(modelDirectories) {
			end = len(modelDirectories)
		}
		batch := modelDirectories[i:end]

		affected, err := r.upsertBatch(batch)
		if err != nil {
			return totalAffected, err
		}
		totalAffected += affected
	}

	return totalAffected, nil
}

// upsertBatch upserts a single batch of directories
func (r *DirectoryRepository) upsertBatch(directories []model.Directory) (int64, error) {
	if len(directories) == 0 {
		return 0, nil
	}

	result := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "url"}, {Name: "target_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"status":         gorm.Expr("COALESCE(EXCLUDED.status, directory.status)"),
			"content_length": gorm.Expr("COALESCE(EXCLUDED.content_length, directory.content_length)"),
			"content_type":   gorm.Expr("COALESCE(NULLIF(EXCLUDED.content_type, ''), directory.content_type)"),
			"duration":       gorm.Expr("COALESCE(EXCLUDED.duration, directory.duration)"),
		}),
	}).Create(&directories)

	return result.RowsAffected, result.Error
}
