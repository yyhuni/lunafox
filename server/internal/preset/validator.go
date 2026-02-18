package preset

import (
	"fmt"

	"github.com/yyhuni/lunafox/server/internal/engineschema"
	"gopkg.in/yaml.v3"
)

// ValidateConfiguration validates the configuration YAML against all applicable schemas.
// It extracts top-level keys from the configuration and validates each one that has a schema.
func ValidateConfiguration(configuration string) error {
	if configuration == "" {
		return nil
	}

	// Parse configuration to extract top-level keys
	var configMap map[string]any
	if err := yaml.Unmarshal([]byte(configuration), &configMap); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	// Get all available engines dynamically from schema files
	knownEngines, err := engineschema.ListEngines()
	if err != nil {
		return fmt.Errorf("list engines: %w", err)
	}

	// Validate each engine that exists in the configuration
	for _, engine := range knownEngines {
		if _, exists := configMap[engine]; exists {
			if err := engineschema.ValidateYAML(engine, []byte(configuration)); err != nil {
				return fmt.Errorf("engine %s: %w", engine, err)
			}
		}
	}

	return nil
}
