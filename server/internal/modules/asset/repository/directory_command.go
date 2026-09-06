package repository

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm/clause"
)

// BatchCreate creates multiple directories, ignoring duplicates
func (r *DirectoryRepository) BatchCreate(directories []assetdomain.Directory) (int, error) {
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

// BatchDelete deletes multiple directories by IDs
func (r *DirectoryRepository) BatchDelete(ids []int) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.Where("id IN ?", ids).Delete(&model.Directory{})
	return result.RowsAffected, result.Error
}

// BatchUpsert creates or updates multiple directories
func (r *DirectoryRepository) BatchUpsert(directories []assetdomain.Directory) (int64, error) {
	return r.BatchUpsertContext(context.Background(), directories)
}

func (r *DirectoryRepository) BatchUpsertContext(ctx context.Context, directories []assetdomain.Directory) (int64, error) {
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

		affected, err := r.upsertBatchContext(ctx, batch)
		if err != nil {
			return totalAffected, err
		}
		totalAffected += affected
	}

	return totalAffected, nil
}

// upsertBatch upserts a single batch of directories
func (r *DirectoryRepository) upsertBatch(directories []model.Directory) (int64, error) {
	return r.upsertBatchContext(context.Background(), directories)
}

func (r *DirectoryRepository) upsertBatchContext(ctx context.Context, directories []model.Directory) (int64, error) {
	if len(directories) == 0 {
		return 0, nil
	}

	result := dbtx.Resolve(ctx, r.db).WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "url"}, {Name: "target_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "content_length", "content_type", "duration"}),
	}).Create(&directories)

	return result.RowsAffected, result.Error
}
