package repository

import (
	"context"
	"strings"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// GetActiveByID finds a target by ID (excluding soft deleted).
func (r *TargetRepository) GetActiveByID(id int) (*catalogdomain.Target, error) {
	return r.GetActiveByIDContext(context.Background(), id)
}

// GetActiveByIDContext preserves a caller-owned cancellation/deadline through the lookup.
func (r *TargetRepository) GetActiveByIDContext(ctx context.Context, id int) (*catalogdomain.Target, error) {
	var target model.Target
	err := dbtx.Resolve(ctx, r.db).WithContext(ctx).Scopes(scope.WithNotDeleted()).
		Where("id = ?", id).
		First(&target).Error
	if err != nil {
		return nil, err
	}
	return targetModelToDomain(&target), nil
}

// List finds all targets with pagination and filters (excluding soft deleted).
func (r *TargetRepository) List(page, pageSize int, filter, orderBy string) ([]catalogdomain.Target, int64, error) {
	return r.ListContext(context.Background(), page, pageSize, filter, orderBy)
}

// ListContext preserves a caller-owned cancellation/deadline through target list queries.
func (r *TargetRepository) ListContext(ctx context.Context, page, pageSize int, filter, orderBy string) ([]catalogdomain.Target, int64, error) {
	var targets []model.Target
	var total int64

	db := dbtx.Resolve(ctx, r.db).WithContext(ctx)
	baseQuery := db.Model(&model.Target{}).Scopes(scope.WithNotDeleted())
	baseQuery = applyTargetFilter(baseQuery, filter)

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := baseQuery.Preload("Organizations", "deleted_at IS NULL")
	query = applyTargetOrder(query, orderBy).
		Scopes(scope.WithPagination(page, pageSize))
	if err := query.Find(&targets).Error; err != nil {
		return nil, 0, err
	}

	return targetModelListToDomain(targets), total, nil
}

func applyTargetFilter(db *gorm.DB, filter string) *gorm.DB {
	trimmed := strings.TrimSpace(filter)
	if trimmed == "" {
		return db
	}
	for _, group := range scope.ParseFilter(trimmed) {
		if strings.EqualFold(group.Filter.Field, "type") && group.Filter.Operator != "==" {
			return db.Where("1 = 0")
		}
	}
	return db.Scopes(scope.WithFilterDefault(trimmed, targetFilterMappingNormalized, "displayName"))
}

func applyTargetOrder(db *gorm.DB, orderBy string) *gorm.DB {
	switch strings.TrimSpace(orderBy) {
	case "displayName", "displayName asc":
		return db.Order("name ASC").Order("id ASC")
	case "displayName desc":
		return db.Order("name DESC").Order("id DESC")
	case "createdAt", "createdAt asc":
		return db.Order("created_at ASC").Order("id ASC")
	case "createdAt desc", "":
		return db.Order("created_at DESC").Order("id DESC")
	case "lastScannedAt", "lastScannedAt asc":
		return db.Order("last_scanned_at ASC NULLS LAST").Order("id ASC")
	case "lastScannedAt desc":
		return db.Order("last_scanned_at DESC NULLS LAST").Order("id DESC")
	default:
		return db.Where("1 = 0")
	}
}

// ExistsByName checks if target name exists (excluding soft deleted).
func (r *TargetRepository) ExistsByName(name string, excludeID ...int) (bool, error) {
	var count int64
	query := r.db.Model(&model.Target{}).
		Scopes(scope.WithNotDeleted()).
		Where("name = ?", name)
	if len(excludeID) > 0 {
		query = query.Where("id != ?", excludeID[0])
	}

	err := query.Count(&count).Error
	return count > 0, err
}

// FindByNames finds targets by names (excluding soft deleted).
func (r *TargetRepository) FindByNames(names []string) ([]catalogdomain.Target, error) {
	return r.FindByNamesContext(context.Background(), names)
}

// FindByNamesContext finds active targets while preserving the ambient transaction.
func (r *TargetRepository) FindByNamesContext(ctx context.Context, names []string) ([]catalogdomain.Target, error) {
	if len(names) == 0 {
		return []catalogdomain.Target{}, nil
	}

	var targets []model.Target
	err := dbtx.Resolve(ctx, r.db).WithContext(ctx).Scopes(scope.WithNotDeleted()).
		Where("name IN ?", names).
		Find(&targets).Error
	if err != nil {
		return nil, err
	}

	return targetModelListToDomain(targets), nil
}
