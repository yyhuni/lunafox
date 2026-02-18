package repository

import (
	"database/sql"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// FindByTargetID finds subdomains by target ID with pagination and filter
func (r *SubdomainRepository) FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Subdomain, int64, error) {
	var subdomains []model.Subdomain
	var total int64

	baseQuery := r.db.Model(&model.Subdomain{}).Where("target_id = ?", targetID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, subdomainFilterMappingNormalized, "name"))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.Scopes(
		scope.WithPagination(page, pageSize),
		scope.OrderByCreatedAtDesc(),
	).Find(&subdomains).Error
	if err != nil {
		return nil, 0, err
	}

	return subdomainModelListToDomain(subdomains), total, nil
}

// StreamByTargetID returns a sql.Rows cursor for streaming export
func (r *SubdomainRepository) StreamByTargetID(targetID int) (*sql.Rows, error) {
	return r.db.Model(&model.Subdomain{}).
		Where("target_id = ?", targetID).
		Order("created_at DESC").
		Rows()
}

// CountByTargetID returns the count of subdomains for a target
func (r *SubdomainRepository) CountByTargetID(targetID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.Subdomain{}).Where("target_id = ?", targetID).Count(&count).Error
	return count, err
}

// ScanRow scans a single row into Subdomain domain object
func (r *SubdomainRepository) ScanRow(rows *sql.Rows) (*assetdomain.Subdomain, error) {
	var subdomain model.Subdomain
	if err := r.db.ScanRows(rows, &subdomain); err != nil {
		return nil, err
	}
	return subdomainModelToDomain(&subdomain), nil
}
