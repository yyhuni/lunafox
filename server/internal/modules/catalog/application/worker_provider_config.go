package application

import (
	"errors"
	"fmt"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"gopkg.in/yaml.v3"
)

var (
	ErrWorkerScanNotFound                   = catalogdomain.ErrWorkerScanNotFound
	ErrWorkerToolRequired                   = catalogdomain.ErrWorkerToolRequired
	ErrWorkerProviderConfigSettingsNotFound = errors.New("worker provider config settings not found")
)

type WorkerProviderConfigScanGuard interface {
	EnsureActiveByID(id int) error
}

type WorkerProviderConfigSettingsStore interface {
	GetInstance() (*catalogdomain.SubfinderProviderSettings, error)
}

type WorkerProviderConfigService struct {
	scanGuard     WorkerProviderConfigScanGuard
	settingsStore WorkerProviderConfigSettingsStore
}

func NewWorkerProviderConfigService(scanGuard WorkerProviderConfigScanGuard, settingsStore WorkerProviderConfigSettingsStore) *WorkerProviderConfigService {
	return &WorkerProviderConfigService{
		scanGuard:     scanGuard,
		settingsStore: settingsStore,
	}
}

func (service *WorkerProviderConfigService) GetProviderConfig(scanID int, toolName string) (string, error) {
	normalizedToolName := catalogdomain.NormalizeSubfinderToolName(toolName)
	if normalizedToolName == "" {
		return "", ErrWorkerToolRequired
	}

	if err := service.scanGuard.EnsureActiveByID(scanID); err != nil {
		return "", err
	}

	return service.generateProviderConfig(normalizedToolName)
}

func (service *WorkerProviderConfigService) generateProviderConfig(toolName string) (string, error) {
	switch toolName {
	case "subfinder":
		return service.generateSubfinderConfig()
	default:
		return "", nil
	}
}

func (service *WorkerProviderConfigService) generateSubfinderConfig() (string, error) {
	settings, err := service.settingsStore.GetInstance()
	if err != nil {
		if errors.Is(err, ErrWorkerProviderConfigSettingsNotFound) {
			return "", nil
		}
		return "", err
	}

	config := make(map[string][]string)
	hasEnabled := false

	for providerName, formatInfo := range catalogdomain.SubfinderProviderFormats {
		providerConfig, exists := settings.Providers[providerName]
		if !exists || !providerConfig.Enabled {
			config[providerName] = []string{}
			continue
		}

		value := catalogdomain.BuildSubfinderProviderCredentialValue(
			formatInfo.Type,
			formatInfo.Format,
			providerConfig.Email,
			providerConfig.APIKey,
			providerConfig.APIID,
			providerConfig.APISecret,
		)
		if value != "" {
			config[providerName] = []string{value}
			hasEnabled = true
		} else {
			config[providerName] = []string{}
		}
	}

	if !hasEnabled {
		return "", nil
	}

	yamlBytes, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal provider config: %w", err)
	}

	return string(yamlBytes), nil
}
