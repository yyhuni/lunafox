package repository

import (
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
)

// GetInstance returns the singleton settings (id=1)
func (r *SubfinderProviderSettingsRepository) GetInstance() (*catalogdomain.SubfinderProviderSettings, error) {
	var settings model.SubfinderProviderSettings
	if err := r.db.First(&settings, 1).Error; err != nil {
		return nil, err
	}
	return subfinderProviderSettingsModelToDomain(&settings), nil
}
