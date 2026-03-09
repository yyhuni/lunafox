package workflowschema

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

func ValidateConfigMap(workflowID string, config map[string]any) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	schema, err := getSchema(workflowID)
	if err != nil {
		return err
	}

	raw, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	var normalized any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return fmt.Errorf("unmarshal config as json: %w", err)
	}

	if err := schema.Validate(normalized); err != nil {
		return fmt.Errorf("invalid %s config: %w", workflowID, err)
	}
	return nil
}

func ValidateYAML(workflowID string, yamlBytes []byte) error {
	if len(yamlBytes) == 0 {
		return fmt.Errorf("yaml is required")
	}

	var root map[string]any
	if err := yaml.Unmarshal(yamlBytes, &root); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}
	if root == nil {
		return fmt.Errorf("yaml must be a mapping")
	}

	config := root
	if nested, ok := root[workflowID]; ok {
		mapping, ok := nested.(map[string]any)
		if !ok {
			return fmt.Errorf("workflow %q config must be a mapping", workflowID)
		}
		config = mapping
	}

	return ValidateConfigMap(workflowID, config)
}
