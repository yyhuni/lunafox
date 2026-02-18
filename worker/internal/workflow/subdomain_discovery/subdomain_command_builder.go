package subdomain_discovery

import "github.com/yyhuni/lunafox/worker/internal/activity"

// normalizeToolConfig maps external config_schema.key values to internal parameter names.
func normalizeToolConfig(toolName string, config map[string]any) (map[string]any, error) {
	tmpl, err := getTemplate(toolName)
	if err != nil {
		return nil, err
	}
	return activity.MapConfigKeys(tmpl, config)
}

// buildCommand gets the template and builds the command string.
// config must be normalized (internal parameter names).
func buildCommand(toolName string, params map[string]any, config map[string]any) (string, error) {
	tmpl, err := getTemplate(toolName)
	if err != nil {
		return "", err
	}
	builder := activity.NewCommandBuilder()
	return builder.Build(tmpl, params, config)
}
