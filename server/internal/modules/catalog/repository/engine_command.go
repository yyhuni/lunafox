package repository

import (
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
)

// Create creates a new engine.
func (r *EngineRepository) Create(engine *catalogdomain.ScanEngine) error {
	modelEngine := engineDomainToModel(engine)
	if err := r.db.Create(modelEngine).Error; err != nil {
		return err
	}
	if engine != nil {
		*engine = *engineModelToDomain(modelEngine)
	}
	return nil
}

// Update updates an engine.
func (r *EngineRepository) Update(engine *catalogdomain.ScanEngine) error {
	return r.db.Save(engineDomainToModel(engine)).Error
}

// Delete deletes an engine.
func (r *EngineRepository) Delete(id int) error {
	return r.db.Delete(&model.ScanEngine{}, id).Error
}
