package repository

import (
	"database/sql"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// FindByTargetID finds websites by target ID with pagination and filter
func (r *WebsiteRepository) FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Website, int64, error) {
	var websites []model.Website
	var total int64

	baseQuery := r.db.Model(&model.Website{}).Where("target_id = ?", targetID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, websiteFilterMappingNormalized, "url"))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.Scopes(
		scope.WithPagination(page, pageSize),
		scope.OrderByCreatedAtDesc(),
	).Find(&websites).Error
	if err != nil {
		return nil, 0, err
	}

	return websiteModelListToDomain(websites), total, nil
}

// GetByID finds a website by ID
func (r *WebsiteRepository) GetByID(id int) (*assetdomain.Website, error) {
	var website model.Website
	err := r.db.First(&website, id).Error
	if err != nil {
		return nil, err
	}
	return websiteModelToDomain(&website), nil
}

// StreamByTargetID returns a sql.Rows cursor for streaming export
func (r *WebsiteRepository) StreamByTargetID(targetID int) (*sql.Rows, error) {
	return r.db.Model(&model.Website{}).
		Where("target_id = ?", targetID).
		Order("created_at DESC").
		Rows()
}

// CountByTargetID returns the count of websites for a target
func (r *WebsiteRepository) CountByTargetID(targetID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.Website{}).Where("target_id = ?", targetID).Count(&count).Error
	return count, err
}

// ScanRow scans a single row into Website domain object
func (r *WebsiteRepository) ScanRow(rows *sql.Rows) (*assetdomain.Website, error) {
	var website model.Website
	if err := r.db.ScanRows(rows, &website); err != nil {
		return nil, err
	}
	return websiteModelToDomain(&website), nil
}
