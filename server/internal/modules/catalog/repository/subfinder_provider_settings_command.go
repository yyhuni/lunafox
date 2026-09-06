package repository

import (
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

// Update updates the settings.
func (r *SubfinderProviderSettingsRepository) Update(settings *catalogdomain.SubfinderProviderSettings) error {
	modelSettings := subfinderProviderSettingsDomainToModel(settings)
	modelSettings.ID = 1 // Force singleton
	return r.db.Save(modelSettings).Error
}
