package application

import (
	"errors"
	"fmt"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

var ErrSubfinderProviderSettingsNotFound = errors.New("subfinder provider settings not found")
var ErrUnsupportedSubfinderProvider = errors.New("unsupported subfinder provider")
var ErrInvalidSubfinderProviderSettings = errors.New("invalid subfinder provider settings")

type SubfinderAPIKeySettingsStore interface {
	GetInstance() (*catalogdomain.SubfinderProviderSettings, error)
	Update(settings *catalogdomain.SubfinderProviderSettings) error
}

type SubfinderAPIKeySettingsService struct {
	store SubfinderAPIKeySettingsStore
}

func NewSubfinderAPIKeySettingsService(store SubfinderAPIKeySettingsStore) *SubfinderAPIKeySettingsService {
	return &SubfinderAPIKeySettingsService{store: store}
}

func (service *SubfinderAPIKeySettingsService) GetSettings() (*catalogdomain.SubfinderProviderSettings, error) {
	settings, err := service.loadSettings()
	if err != nil {
		return nil, err
	}
	return cloneSubfinderProviderSettings(settings), nil
}

func (service *SubfinderAPIKeySettingsService) UpdateSettings(providers catalogdomain.SubfinderProviderConfigs) (*catalogdomain.SubfinderProviderSettings, error) {
	if err := validateSubfinderProviderNames(providers); err != nil {
		return nil, err
	}

	settings, err := service.loadSettings()
	if err != nil {
		return nil, err
	}

	for providerName, requestedConfig := range providers {
		currentConfig := settings.Providers[providerName]
		mergedConfig := mergeSubfinderProviderConfig(currentConfig, requestedConfig)
		if err := validateSubfinderProviderConfig(providerName, mergedConfig); err != nil {
			return nil, err
		}
		mergedConfig.Status = statusAfterSubfinderProviderUpdate(currentConfig, mergedConfig)
		settings.Providers[providerName] = mergedConfig
	}
	settings.ID = 1

	if err := service.store.Update(settings); err != nil {
		return nil, err
	}

	return cloneSubfinderProviderSettings(settings), nil
}

func (service *SubfinderAPIKeySettingsService) loadSettings() (*catalogdomain.SubfinderProviderSettings, error) {
	settings, err := service.store.GetInstance()
	if err != nil {
		if errors.Is(err, ErrSubfinderProviderSettingsNotFound) {
			return newDefaultSubfinderProviderSettings(), nil
		}
		return nil, err
	}

	settings = cloneSubfinderProviderSettings(settings)
	ensureSupportedSubfinderProviders(settings.Providers)
	settings.ID = 1
	return settings, nil
}

func validateSubfinderProviderNames(providers catalogdomain.SubfinderProviderConfigs) error {
	for providerName := range providers {
		if _, ok := catalogdomain.LookupSubfinderProviderDefinition(providerName); !ok {
			return fmt.Errorf("%w: %s", ErrUnsupportedSubfinderProvider, providerName)
		}
	}
	return nil
}

// mergeSubfinderProviderConfig applies field-level presence semantics: an omitted
// value keeps the stored credential, while an explicit empty value clears it.
func mergeSubfinderProviderConfig(
	current catalogdomain.SubfinderProviderConfig,
	requested catalogdomain.SubfinderProviderConfig,
) catalogdomain.SubfinderProviderConfig {
	values := make(map[string]string, len(current.Values)+len(requested.Values))
	for fieldName, value := range current.Values {
		values[fieldName] = value
	}
	for fieldName, value := range requested.Values {
		values[fieldName] = value
	}

	return catalogdomain.SubfinderProviderConfig{
		Enabled: requested.Enabled,
		Values:  values,
		// The client cannot select status; migration statuses are carried forward
		// until a valid enabled configuration replaces them.
		Status: current.Status,
	}
}

func statusAfterSubfinderProviderUpdate(
	current catalogdomain.SubfinderProviderConfig,
	merged catalogdomain.SubfinderProviderConfig,
) string {
	if current.Status == catalogdomain.SubfinderProviderStatusUnsupported {
		return current.Status
	}
	if current.Status == catalogdomain.SubfinderProviderStatusRequiresReconfiguration && !merged.Enabled {
		return current.Status
	}
	if merged.Enabled {
		return catalogdomain.SubfinderProviderStatusConfigured
	}
	return catalogdomain.SubfinderProviderStatusUnconfigured
}

func validateSubfinderProviderConfig(providerName string, providerConfig catalogdomain.SubfinderProviderConfig) error {
	if !providerConfig.Enabled {
		return nil
	}
	definition, _ := catalogdomain.LookupSubfinderProviderDefinition(providerName)
	for _, field := range definition.Fields {
		if field.Required && providerConfig.Values[field.Name] == "" {
			return fmt.Errorf("%w: %s missing %s", ErrInvalidSubfinderProviderSettings, providerName, field.Name)
		}
	}
	if catalogdomain.BuildSubfinderProviderCredentialValue(providerName, providerConfig) == "" {
		return fmt.Errorf("%w: %s credential format", ErrInvalidSubfinderProviderSettings, providerName)
	}
	return nil
}

func newDefaultSubfinderProviderSettings() *catalogdomain.SubfinderProviderSettings {
	providers := make(catalogdomain.SubfinderProviderConfigs, len(catalogdomain.SubfinderProviderDefinitions()))
	ensureSupportedSubfinderProviders(providers)
	return &catalogdomain.SubfinderProviderSettings{ID: 1, Providers: providers}
}

func ensureSupportedSubfinderProviders(providers catalogdomain.SubfinderProviderConfigs) {
	for _, definition := range catalogdomain.SubfinderProviderDefinitions() {
		providerName := definition.Key
		if _, ok := providers[providerName]; !ok {
			providers[providerName] = catalogdomain.SubfinderProviderConfig{Status: catalogdomain.SubfinderProviderStatusUnconfigured, Values: map[string]string{}}
			continue
		}
		providerConfig := providers[providerName]
		if providerConfig.Values == nil {
			providerConfig.Values = map[string]string{}
		}
		if providerConfig.Status == "" {
			if providerConfig.Enabled {
				providerConfig.Status = catalogdomain.SubfinderProviderStatusConfigured
			} else {
				providerConfig.Status = catalogdomain.SubfinderProviderStatusUnconfigured
			}
		}
		providers[providerName] = providerConfig
	}
}

func cloneSubfinderProviderSettings(settings *catalogdomain.SubfinderProviderSettings) *catalogdomain.SubfinderProviderSettings {
	if settings == nil {
		return newDefaultSubfinderProviderSettings()
	}
	providers := make(catalogdomain.SubfinderProviderConfigs, len(settings.Providers))
	for providerName, providerConfig := range settings.Providers {
		values := make(map[string]string, len(providerConfig.Values))
		for fieldName, value := range providerConfig.Values {
			values[fieldName] = value
		}
		providerConfig.Values = values
		providers[providerName] = providerConfig
	}
	ensureSupportedSubfinderProviders(providers)
	return &catalogdomain.SubfinderProviderSettings{ID: settings.ID, Providers: providers}
}
