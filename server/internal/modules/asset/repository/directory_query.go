package repository

import (
	"database/sql"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// FindByTargetID finds directories by target ID with pagination and filter
func (r *DirectoryRepository) FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Directory, int64, error) {
	var directories []model.Directory
	var total int64

	baseQuery := r.db.Model(&model.Directory{}).Where("target_id = ?", targetID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, directoryFilterMappingNormalized, "url"))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.Scopes(
		scope.WithPagination(page, pageSize),
		scope.OrderByCreatedAtDesc(),
	).Find(&directories).Error
	if err != nil {
		return nil, 0, err
	}

	return directoryModelListToDomain(directories), total, nil
}

// StreamByTargetID returns a sql.Rows cursor for streaming export
func (r *DirectoryRepository) StreamByTargetID(targetID int) (*sql.Rows, error) {
	return r.db.Model(&model.Directory{}).
		Where("target_id = ?", targetID).
		Order("created_at DESC").
		Rows()
}

// CountByTargetID returns the count of directories for a target
func (r *DirectoryRepository) CountByTargetID(targetID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.Directory{}).Where("target_id = ?", targetID).Count(&count).Error
	return count, err
}

// ScanRow scans a single row into Directory domain object
func (r *DirectoryRepository) ScanRow(rows *sql.Rows) (*assetdomain.Directory, error) {
	var directory model.Directory
	if err := r.db.ScanRows(rows, &directory); err != nil {
		return nil, err
	}
	return directoryModelToDomain(&directory), nil
}
