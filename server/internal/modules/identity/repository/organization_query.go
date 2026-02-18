package repository

import (
	"github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// FindByID finds an organization by ID (excluding soft deleted).
func (r *OrganizationRepository) GetActiveByID(id int) (*model.Organization, error) {
	var org model.Organization
	err := r.db.Scopes(scope.WithNotDeleted()).
		Where("id = ?", id).
		First(&org).Error
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// FindByIDWithCount finds an organization by ID with target count (excluding soft deleted).
func (r *OrganizationRepository) FindByIDWithCount(id int) (*OrganizationWithCount, error) {
	var org OrganizationWithCount
	err := r.db.Table("organization").
		Select(`organization.*,
			(SELECT COUNT(*) FROM organization_target
			 INNER JOIN target ON target.id = organization_target.target_id
			 WHERE organization_target.organization_id = organization.id
			 AND target.deleted_at IS NULL) as target_count`).
		Where("organization.id = ? AND organization.deleted_at IS NULL", id).
		First(&org).Error
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// FindAll finds all organizations with pagination and target count (excluding soft deleted).
func (r *OrganizationRepository) FindAll(page, pageSize int, filter string) ([]OrganizationWithCount, int64, error) {
	var orgs []OrganizationWithCount
	var total int64

	countQuery := r.db.Model(&model.Organization{}).
		Scopes(scope.WithNotDeleted()).
		Scopes(scope.WithFilterDefault(filter, organizationFilterMappingNormalized, "name"))
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := r.db.Table("organization").
		Select(`organization.*,
			(SELECT COUNT(*) FROM organization_target
			 INNER JOIN target ON target.id = organization_target.target_id
			 WHERE organization_target.organization_id = organization.id
			 AND target.deleted_at IS NULL) as target_count`).
		Where("organization.deleted_at IS NULL").
		Scopes(scope.WithFilterDefault(filter, organizationFilterMappingNormalized, "name"))

	err := query.Scopes(
		scope.WithPagination(page, pageSize),
		scope.OrderBy("organization.created_at", true),
	).Find(&orgs).Error

	return orgs, total, err
}

// ExistsByName checks if organization name exists (excluding soft deleted).
func (r *OrganizationRepository) ExistsByName(name string, excludeID ...int) (bool, error) {
	var count int64
	query := r.db.Model(&model.Organization{}).
		Scopes(scope.WithNotDeleted()).
		Where("name = ?", name)
	if len(excludeID) > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	err := query.Count(&count).Error
	return count > 0, err
}

// Exists checks if organization exists by ID (excluding soft deleted).
func (r *OrganizationRepository) Exists(id int) (bool, error) {
	var count int64
	err := r.db.Model(&model.Organization{}).
		Scopes(scope.WithNotDeleted()).
		Where("id = ?", id).
		Count(&count).Error
	return count > 0, err
}

// FindTargets finds targets belonging to an organization with pagination.
func (r *OrganizationRepository) FindTargets(organizationID int, page, pageSize int, targetType, filter string) ([]model.OrganizationTargetRef, int64, error) {
	var targets []model.OrganizationTargetRef
	var total int64

	targetFilterMapping := scope.NormalizeFilterMapping(scope.FilterMapping{
		"name": {Column: "target.name", IsArray: false},
	})

	query := r.db.Model(&model.OrganizationTargetRef{}).
		Joins("INNER JOIN organization_target ON organization_target.target_id = target.id").
		Where("organization_target.organization_id = ? AND target.deleted_at IS NULL", organizationID)

	if targetType != "" {
		query = query.Where("target.type = ?", targetType)
	}

	query = query.Scopes(scope.WithFilterDefault(filter, targetFilterMapping, "name"))

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Scopes(
		scope.WithPagination(page, pageSize),
		scope.OrderBy("target.created_at", true),
	).Find(&targets).Error

	return targets, total, err
}
