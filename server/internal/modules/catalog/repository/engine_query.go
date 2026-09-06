package repository

import (
	"errors"
	"fmt"
	"strings"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/gorm"
)

func (repository *EngineRepository) ListInstalledEngines() ([]catalogdomain.Engine, error) {
	var records []model.Engine
	if err := repository.db.Order("engine_id ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]catalogdomain.Engine, 0, len(records))
	for index := range records {
		result = append(result, *engineModelToDomain(&records[index]))
	}
	return result, nil
}

func (repository *EngineRepository) GetInstalledEngineByID(engineID string) (*catalogdomain.Engine, error) {
	var record model.Engine
	if err := repository.db.Where("engine_id = ?", strings.TrimSpace(engineID)).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %q", catalogdomain.ErrEngineNotFound, strings.TrimSpace(engineID))
		}
		return nil, err
	}
	return engineModelToDomain(&record), nil
}
