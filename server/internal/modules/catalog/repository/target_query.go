package repository

import (
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// GetActiveByID finds a target by ID (excluding soft deleted).
func (r *TargetRepository) GetActiveByID(id int) (*catalogdomain.Target, error) {
	var target model.Target
	err := r.db.Scopes(scope.WithNotDeleted()).
		Where("id = ?", id).
		First(&target).Error
	if err != nil {
		return nil, err
	}
	return targetModelToDomain(&target), nil
}

// FindAll finds all targets with pagination and filters (excluding soft deleted).
func (r *TargetRepository) FindAll(page, pageSize int, targetType, filter string) ([]catalogdomain.Target, int64, error) {
	var targets []model.Target
	var total int64

	baseQuery := r.db.Model(&model.Target{}).Scopes(scope.WithNotDeleted())
	if targetType != "" {
		baseQuery = baseQuery.Where("type = ?", targetType)
	}
	if filter != "" {
		baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, targetFilterMappingNormalized, "name"))
	}

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.
		Preload("Organizations", "deleted_at IS NULL").
		Scopes(
			scope.WithPagination(page, pageSize),
			scope.OrderByCreatedAtDesc(),
		).
		Find(&targets).Error
	if err != nil {
		return nil, 0, err
	}

	return targetModelListToDomain(targets), total, nil
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
	if len(names) == 0 {
		return []catalogdomain.Target{}, nil
	}

	var targets []model.Target
	err := r.db.Scopes(scope.WithNotDeleted()).
		Where("name IN ?", names).
		Find(&targets).Error
	if err != nil {
		return nil, err
	}

	return targetModelListToDomain(targets), nil
}
