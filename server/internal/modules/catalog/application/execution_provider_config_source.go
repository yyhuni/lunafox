package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"gopkg.in/yaml.v3"
)

var ErrExecutionProviderConfigInvalid = errors.New("execution provider config is invalid")

// ExecutionSubfinderProviderSettingsStore is the cancellation-aware settings
// read boundary used only by pre-start execution artifact production.
type ExecutionSubfinderProviderSettingsStore interface {
	GetInstanceContext(ctx context.Context) (*catalogdomain.SubfinderProviderSettings, error)
}

// ExecutionProviderConfigSource produces late-bound provider configuration
// under the same context as the requesting artifact stream.
type ExecutionProviderConfigSource struct {
	settingsStore ExecutionSubfinderProviderSettingsStore
}

// NewExecutionProviderConfigSource binds execution-only provider reads to a cancellation-aware store.
func NewExecutionProviderConfigSource(settingsStore ExecutionSubfinderProviderSettingsStore) *ExecutionProviderConfigSource {
	return &ExecutionProviderConfigSource{settingsStore: settingsStore}
}

// GetExecutionSubfinderProviderConfig returns the complete late-bound provider
// mapping for one execution artifact attempt and preserves caller cancellation.
func (source *ExecutionProviderConfigSource) GetExecutionSubfinderProviderConfig(ctx context.Context) (string, error) {
	if source == nil || source.settingsStore == nil {
		return "", fmt.Errorf("%w: subfinder provider settings store is required", ErrExecutionProviderConfigInvalid)
	}
	if ctx == nil {
		return "", fmt.Errorf("%w: context is required", ErrExecutionProviderConfigInvalid)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	settings, err := source.settingsStore.GetInstanceContext(ctx)
	if err != nil {
		if errors.Is(err, ErrSubfinderProviderSettingsNotFound) {
			return "", fmt.Errorf("%w: %v", ErrExecutionProviderConfigInvalid, err)
		}
		return "", err
	}
	config, err := projectSubfinderProviderConfig(settings)
	if err != nil {
		return "", err
	}
	content, err := marshalSubfinderProviderConfig(config)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrExecutionProviderConfigInvalid, err)
	}
	if err := validateCompleteSubfinderProviderConfig(content); err != nil {
		return "", fmt.Errorf("%w: %v", ErrExecutionProviderConfigInvalid, err)
	}
	return content, nil
}

func projectSubfinderProviderConfig(settings *catalogdomain.SubfinderProviderSettings) (map[string][]string, error) {
	if settings == nil {
		return nil, fmt.Errorf("%w: subfinder provider settings are required", ErrExecutionProviderConfigInvalid)
	}

	config := make(map[string][]string, len(catalogdomain.SubfinderProviderDefinitions()))
	for _, definition := range catalogdomain.SubfinderProviderDefinitions() {
		providerConfig, exists := settings.Providers[definition.Key]
		if !exists {
			config[definition.SourceName] = []string{}
			continue
		}
		value := catalogdomain.BuildSubfinderProviderCredentialValue(definition.Key, providerConfig)
		if value == "" {
			config[definition.SourceName] = []string{}
			continue
		}
		config[definition.SourceName] = []string{value}
	}
	return config, nil
}

func marshalSubfinderProviderConfig(config map[string][]string) (string, error) {
	yamlBytes, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal provider config: %w", err)
	}
	return string(yamlBytes), nil
}

func validateCompleteSubfinderProviderConfig(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("subfinder provider config must be non-empty")
	}
	var decoded map[string][]string
	if err := yaml.Unmarshal([]byte(content), &decoded); err != nil {
		return fmt.Errorf("validate subfinder provider config: %w", err)
	}
	definitions := catalogdomain.SubfinderProviderDefinitions()
	if len(decoded) != len(definitions) {
		return fmt.Errorf("subfinder provider config key set does not match registry %s", catalogdomain.SubfinderProviderRegistryVersion)
	}
	for _, definition := range definitions {
		if values, ok := decoded[definition.SourceName]; !ok || values == nil {
			return fmt.Errorf("subfinder provider config key %q must contain a string array", definition.SourceName)
		}
	}
	return nil
}
