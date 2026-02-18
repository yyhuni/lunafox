package repository

import (
	"database/sql"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// FindByTargetID finds endpoints by target ID with pagination and filter
func (r *EndpointRepository) FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Endpoint, int64, error) {
	var endpoints []model.Endpoint
	var total int64

	baseQuery := r.db.Model(&model.Endpoint{}).Where("target_id = ?", targetID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, endpointFilterMappingNormalized, "url"))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.Scopes(
		scope.WithPagination(page, pageSize),
		scope.OrderByCreatedAtDesc(),
	).Find(&endpoints).Error
	if err != nil {
		return nil, 0, err
	}

	return endpointModelListToDomain(endpoints), total, nil
}

// GetByID finds an endpoint by ID
func (r *EndpointRepository) GetByID(id int) (*assetdomain.Endpoint, error) {
	var endpoint model.Endpoint
	err := r.db.First(&endpoint, id).Error
	if err != nil {
		return nil, err
	}
	return endpointModelToDomain(&endpoint), nil
}

// StreamByTargetID returns a sql.Rows cursor for streaming export
func (r *EndpointRepository) StreamByTargetID(targetID int) (*sql.Rows, error) {
	return r.db.Model(&model.Endpoint{}).
		Where("target_id = ?", targetID).
		Order("created_at DESC").
		Rows()
}

// CountByTargetID returns the count of endpoints for a target
func (r *EndpointRepository) CountByTargetID(targetID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.Endpoint{}).Where("target_id = ?", targetID).Count(&count).Error
	return count, err
}

// ScanRow scans a single row into Endpoint domain object
func (r *EndpointRepository) ScanRow(rows *sql.Rows) (*assetdomain.Endpoint, error) {
	var endpoint model.Endpoint
	if err := r.db.ScanRows(rows, &endpoint); err != nil {
		return nil, err
	}
	return endpointModelToDomain(&endpoint), nil
}
