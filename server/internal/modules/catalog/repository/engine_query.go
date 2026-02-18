package repository

import (
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// GetByID finds an engine by ID.
func (r *EngineRepository) GetByID(id int) (*catalogdomain.ScanEngine, error) {
	var engine model.ScanEngine
	err := r.db.First(&engine, id).Error
	if err != nil {
		return nil, err
	}
	return engineModelToDomain(&engine), nil
}

// FindAll finds all engines with pagination.
func (r *EngineRepository) FindAll(page, pageSize int) ([]catalogdomain.ScanEngine, int64, error) {
	var engines []model.ScanEngine
	var total int64

	if err := r.db.Model(&model.ScanEngine{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Scopes(
		scope.WithPagination(page, pageSize),
		scope.OrderByCreatedAtDesc(),
	).Find(&engines).Error
	if err != nil {
		return nil, 0, err
	}

	return engineModelListToDomain(engines), total, nil
}

// ExistsByName checks if engine name exists.
func (r *EngineRepository) ExistsByName(name string, excludeID ...int) (bool, error) {
	var count int64
	query := r.db.Model(&model.ScanEngine{}).Where("name = ?", name)
	if len(excludeID) > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	err := query.Count(&count).Error
	return count > 0, err
}
