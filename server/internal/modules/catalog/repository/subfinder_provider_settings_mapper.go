package repository

import (
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
)

const legacyHunterProviderKey = "hunter_legacy"

func subfinderProviderSettingsModelToDomain(settings *model.SubfinderProviderSettings) *catalogdomain.SubfinderProviderSettings {
	if settings == nil {
		return nil
	}

	providers := make(catalogdomain.SubfinderProviderConfigs, len(settings.Providers))
	for providerName, providerConfig := range settings.Providers {
		mergeMigratedSubfinderProviderConfig(providers, providerName, providerConfig)
	}

	return &catalogdomain.SubfinderProviderSettings{
		ID:        settings.ID,
		Providers: providers,
	}
}

func subfinderProviderSettingsDomainToModel(settings *catalogdomain.SubfinderProviderSettings) *model.SubfinderProviderSettings {
	if settings == nil {
		return nil
	}

	providers := make(model.SubfinderProviderConfigs, len(settings.Providers))
	for providerName, providerConfig := range settings.Providers {
		providers[providerName] = model.SubfinderProviderConfig{
			Enabled: providerConfig.Enabled,
			Status:  providerConfig.Status,
			Values:  cloneStringMap(providerConfig.Values),
		}
	}

	return &model.SubfinderProviderSettings{
		ID:        settings.ID,
		Providers: providers,
	}
}

func mergeMigratedSubfinderProviderConfig(providers catalogdomain.SubfinderProviderConfigs, providerName string, providerConfig model.SubfinderProviderConfig) {
	switch providerName {
	case "fofa":
		providers["fofa"] = registryProviderConfig(providerConfig, map[string]string{"email": providerConfig.Email, "apiKey": legacyAPIKey(providerConfig)})
	case "shodan", "securitytrails", "threatbook", "quake":
		providers[providerName] = registryProviderConfig(providerConfig, map[string]string{"apiKey": legacyAPIKey(providerConfig)})
	case "censys":
		if len(providerConfig.Values) > 0 {
			providers["censys"] = registryProviderConfig(providerConfig, providerConfig.Values)
			return
		}
		providers["censys"] = legacyRequiresReconfiguration(providerConfig, map[string]string{"apiId": providerConfig.APIID, "apiSecret": providerConfig.APISecret})
	case "zoomeyeapi":
		providers["zoomeyeapi"] = registryProviderConfig(providerConfig, providerConfig.Values)
	case "zoomeye":
		providers["zoomeyeapi"] = legacyRequiresReconfiguration(providerConfig, map[string]string{"apiKey": legacyAPIKey(providerConfig)})
	case "hunter":
		providers[legacyHunterProviderKey] = catalogdomain.SubfinderProviderConfig{
			Enabled: false,
			Status:  catalogdomain.SubfinderProviderStatusUnsupported,
			Values:  map[string]string{"apiKey": legacyAPIKey(providerConfig)},
		}
	default:
		if _, ok := catalogdomain.LookupSubfinderProviderDefinition(providerName); ok {
			providers[providerName] = registryProviderConfig(providerConfig, providerConfig.Values)
		}
	}
}

func registryProviderConfig(providerConfig model.SubfinderProviderConfig, fallbackValues map[string]string) catalogdomain.SubfinderProviderConfig {
	values := cloneStringMap(providerConfig.Values)
	if len(values) == 0 {
		values = compactStringMap(fallbackValues)
	}
	status := providerConfig.Status
	if status == "" {
		if providerConfig.Enabled {
			status = catalogdomain.SubfinderProviderStatusConfigured
		} else {
			status = catalogdomain.SubfinderProviderStatusUnconfigured
		}
	}
	return catalogdomain.SubfinderProviderConfig{Enabled: providerConfig.Enabled, Status: status, Values: values}
}

func legacyRequiresReconfiguration(providerConfig model.SubfinderProviderConfig, values map[string]string) catalogdomain.SubfinderProviderConfig {
	if len(providerConfig.Values) > 0 {
		return registryProviderConfig(providerConfig, providerConfig.Values)
	}
	legacyValues := compactStringMap(values)
	if len(legacyValues) == 0 {
		// Fresh-install seeds use the legacy-shaped columns with empty values; they
		// are unconfigured defaults, not credentials that need migration.
		return registryProviderConfig(providerConfig, legacyValues)
	}
	return catalogdomain.SubfinderProviderConfig{
		Enabled: false,
		Status:  catalogdomain.SubfinderProviderStatusRequiresReconfiguration,
		Values:  legacyValues,
	}
}

func legacyAPIKey(providerConfig model.SubfinderProviderConfig) string {
	if providerConfig.APIKey != "" {
		return providerConfig.APIKey
	}
	return providerConfig.AuthenticationToken
}

func compactStringMap(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		if value != "" {
			result[key] = value
		}
	}
	return result
}

func cloneStringMap(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
