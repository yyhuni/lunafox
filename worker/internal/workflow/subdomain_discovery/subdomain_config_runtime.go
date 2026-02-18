package subdomain_discovery

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

var (
	schemaOnce sync.Once
	schema     *jsonschema.Schema
	schemaErr  error
)

func getConfigSchema() (*jsonschema.Schema, error) {
	schemaOnce.Do(func() {
		b, err := templatesFS.ReadFile("schema_generated.json")
		if err != nil {
			schemaErr = fmt.Errorf("read embedded schema: %w", err)
			return
		}

		compiler := jsonschema.NewCompiler()
		// Use an in-memory resource name so no file system access is required.
		if err := compiler.AddResource("schema.json", strings.NewReader(string(b))); err != nil {
			schemaErr = fmt.Errorf("add schema resource: %w", err)
			return
		}
		s, err := compiler.Compile("schema.json")
		if err != nil {
			schemaErr = fmt.Errorf("compile schema: %w", err)
			return
		}
		schema = s
	})

	return schema, schemaErr
}

// validateExplicitConfig validates the workflow config using the generated JSON schema.
// This keeps the runtime validation aligned with templates.yaml (single source of truth).
func validateExplicitConfig(config map[string]any) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	s, err := getConfigSchema()
	if err != nil {
		return err
	}

	// Convert YAML-decoded types (e.g., int) into JSON-native types (e.g., float64)
	// to match JSON Schema validators' expectations.
	raw, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("unmarshal config as json: %w", err)
	}

	if err := s.Validate(v); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return nil
}

// isStageEnabled checks if a stage is enabled in the config.
func isStageEnabled(config map[string]any, stageName string) bool {
	stageConfig, ok := config[stageName].(map[string]any)
	if !ok {
		return false
	}
	enabled, ok := stageConfig["enabled"].(bool)
	return ok && enabled
}

// isToolEnabled checks if a specific tool is enabled within a stage.
func isToolEnabled(stageConfig map[string]any, toolName string) bool {
	toolsConfig, ok := stageConfig["tools"].(map[string]any)
	if !ok {
		return false
	}
	toolConfig, ok := toolsConfig[toolName].(map[string]any)
	if !ok {
		return false
	}
	enabled, ok := toolConfig["enabled"].(bool)
	return ok && enabled
}

// getConfigPath retrieves a nested config section by path.
func getConfigPath(config map[string]any, path string) map[string]any {
	if config == nil {
		return nil
	}
	parts := strings.Split(path, ".")
	current := config
	for _, part := range parts {
		next, ok := current[part].(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}
	return current
}

// getTimeout extracts timeout from tool config.
// Requires an explicit timeout value.
func getTimeout(toolConfig map[string]any) (time.Duration, error) {
	seconds, err := getIntValue(toolConfig, "timeout-runtime")
	if err != nil {
		return 0, fmt.Errorf("timeout: %w", err)
	}
	if seconds <= 0 {
		return 0, fmt.Errorf("timeout must be > 0")
	}
	return time.Duration(seconds) * time.Second, nil
}

// getIntValue extracts an integer value from config.
func getIntValue(config map[string]any, key string) (int, error) {
	if config == nil {
		return 0, fmt.Errorf("%s is required", key)
	}
	raw, ok := config[key]
	if !ok {
		return 0, fmt.Errorf("%s is required", key)
	}
	switch v := raw.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case uint:
		return int(v), nil
	case uint8:
		return int(v), nil
	case uint16:
		return int(v), nil
	case uint32:
		return int(v), nil
	case uint64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("%s must be integer", key)
	}
}

// getStringValue extracts a string value from config with a default.
func getStringValue(config map[string]any, key, defaultValue string) string {
	if config == nil {
		return defaultValue
	}
	if value, ok := config[key].(string); ok && value != "" {
		return value
	}
	return defaultValue
}
