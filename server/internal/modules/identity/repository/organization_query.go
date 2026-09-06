package repository

import (
	"context"
	"strings"

	model "github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// FindByID finds an organization by ID (excluding soft deleted).
func (r *OrganizationRepository) GetActiveByID(id int) (*model.Organization, error) {
	return r.GetActiveByIDContext(context.Background(), id)
}

// GetActiveByIDContext preserves cancellation and participates in an ambient transaction.
func (r *OrganizationRepository) GetActiveByIDContext(ctx context.Context, id int) (*model.Organization, error) {
	var org model.Organization
	err := dbtx.Resolve(ctx, r.db).WithContext(ctx).Scopes(scope.WithNotDeleted()).
		Where("id = ?", id).
		First(&org).Error
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// FindByIDWithCount finds an organization by ID with target count (excluding soft deleted).
func (r *OrganizationRepository) FindByIDWithCount(id int) (*OrganizationWithCount, error) {
	return r.FindByIDWithCountContext(context.Background(), id)
}

// FindByIDWithCountContext preserves cancellation for request-scoped readers.
func (r *OrganizationRepository) FindByIDWithCountContext(ctx context.Context, id int) (*OrganizationWithCount, error) {
	var org OrganizationWithCount
	err := dbtx.Resolve(ctx, r.db).WithContext(ctx).Table("organization").
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
func (r *OrganizationRepository) List(page, pageSize int, filter, orderBy string) ([]OrganizationWithCount, int64, error) {
	return r.ListContext(context.Background(), page, pageSize, filter, orderBy)
}

// ListContext preserves cancellation for request-scoped readers.
func (r *OrganizationRepository) ListContext(ctx context.Context, page, pageSize int, filter, orderBy string) ([]OrganizationWithCount, int64, error) {
	var orgs []OrganizationWithCount
	var total int64

	db := dbtx.Resolve(ctx, r.db).WithContext(ctx)
	countQuery := db.Model(&model.Organization{}).
		Scopes(scope.WithNotDeleted()).
		Scopes(scope.WithFilterDefault(filter, organizationFilterMappingNormalized, "displayName"))
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := db.Table("organization").
		Select(`organization.*,
			(SELECT COUNT(*) FROM organization_target
			 INNER JOIN target ON target.id = organization_target.target_id
			 WHERE organization_target.organization_id = organization.id
			 AND target.deleted_at IS NULL) as target_count`).
		Where("organization.deleted_at IS NULL").
		Scopes(scope.WithFilterDefault(filter, organizationFilterMappingNormalized, "displayName"))

	err := applyOrganizationOrder(query, orderBy).
		Scopes(scope.WithPagination(page, pageSize)).
		Find(&orgs).Error

	return orgs, total, err
}

func applyOrganizationOrder(db *gorm.DB, orderBy string) *gorm.DB {
	switch strings.TrimSpace(orderBy) {
	case "displayName", "displayName asc":
		return db.Order("organization.name ASC").Order("organization.id ASC")
	case "displayName desc":
		return db.Order("organization.name DESC").Order("organization.id DESC")
	case "createdAt", "createdAt asc":
		return db.Order("organization.created_at ASC").Order("organization.id ASC")
	case "createdAt desc", "":
		return db.Order("organization.created_at DESC").Order("organization.id DESC")
	default:
		return db.Where("1 = 0")
	}
}

// ExistsByName checks if organization name exists (excluding soft deleted).
func (r *OrganizationRepository) ExistsByName(name string, excludeID ...int) (bool, error) {
	return r.ExistsByNameContext(context.Background(), name, excludeID...)
}

// ExistsByNameContext preserves cancellation and participates in an ambient transaction.
func (r *OrganizationRepository) ExistsByNameContext(ctx context.Context, name string, excludeID ...int) (bool, error) {
	var count int64
	query := dbtx.Resolve(ctx, r.db).WithContext(ctx).Model(&model.Organization{}).
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
	return r.ExistsContext(context.Background(), id)
}

func (r *OrganizationRepository) ExistsContext(ctx context.Context, id int) (bool, error) {
	var count int64
	err := dbtx.Resolve(ctx, r.db).WithContext(ctx).Model(&model.Organization{}).
		Scopes(scope.WithNotDeleted()).
		Where("id = ?", id).
		Count(&count).Error
	return count > 0, err
}

// FindTargets finds targets belonging to an organization with pagination.
func (r *OrganizationRepository) ListTargetsByOrganizationID(organizationID int, page, pageSize int, targetType, filter string) ([]model.OrganizationTargetRef, int64, error) {
	return r.ListTargetsByOrganizationIDContext(context.Background(), organizationID, page, pageSize, targetType, filter)
}

func (r *OrganizationRepository) ListTargetsByOrganizationIDContext(ctx context.Context, organizationID int, page, pageSize int, targetType, filter string) ([]model.OrganizationTargetRef, int64, error) {
	var targets []model.OrganizationTargetRef
	var total int64

	targetFilterMapping := scope.NormalizeFilterMapping(scope.FilterMapping{
		"name": {Column: "target.name", IsArray: false},
	})

	query := r.db.WithContext(ctx).Model(&model.OrganizationTargetRef{}).
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
