package activity

import "fmt"

// MapConfigKeys maps external config_schema.key values to internal parameter semantic IDs.
//
// It ignores the special key "enabled", merges internal_params, and rejects unknown keys.
func MapConfigKeys(tmpl CommandTemplate, raw map[string]any) (map[string]any, error) {
	normalized := make(map[string]any)
	if tmpl.InternalParams != nil {
		for key, value := range tmpl.InternalParams {
			normalized[key] = value
		}
	}

	configKeyIndex := make(map[string]Parameter, len(tmpl.RuntimeParams)+len(tmpl.CLIParams))
	for _, param := range allParams(tmpl) {
		configKeyIndex[param.ConfigSchema.Key] = param
	}
	if raw == nil {
		return normalized, nil
	}

	for key, value := range raw {
		if key == "enabled" {
			continue
		}
		if _, exists := tmpl.InternalParams[key]; exists {
			return nil, fmt.Errorf("config key %s is internal and cannot be set", key)
		}
		param, ok := configKeyIndex[key]
		if !ok {
			return nil, fmt.Errorf("unknown config key %s", key)
		}
		if err := validateType(value, param.ConfigSchema.Type); err != nil {
			return nil, fmt.Errorf("parameter %s: %w", param.Var, err)
		}
		if param.SemanticID == "" {
			return nil, fmt.Errorf("parameter %s: semantic_id is required", param.Var)
		}
		if _, exists := normalized[param.SemanticID]; exists {
			return nil, fmt.Errorf("parameter %s conflicts with internal_params key %s", param.Var, param.SemanticID)
		}
		normalized[param.SemanticID] = value
	}

	return normalized, nil
}
