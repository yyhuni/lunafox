package workflowdefaulting

import (
	"fmt"
	"math"
	"strings"

	workflowmanifest "github.com/yyhuni/lunafox/server/internal/workflow/manifest"
)

func NormalizeRootConfig(root map[string]any, workflowIDs []string) (map[string]any, error) {
	if root == nil {
		return nil, fmt.Errorf("configuration root must be object")
	}
	normalized := deepCopyMap(root)
	for _, workflowID := range workflowIDs {
		workflowID = strings.TrimSpace(workflowID)
		if workflowID == "" {
			continue
		}
		manifest, err := workflowmanifest.GetManifest(workflowID)
		if err != nil {
			return nil, err
		}
		rawWorkflowConfig, ok := normalized[workflowID]
		if !ok {
			return nil, fmt.Errorf("missing %s config; expected nested configuration under key %q", workflowID, workflowID)
		}
		workflowConfig, ok := rawWorkflowConfig.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("workflow %s configuration must be object", workflowID)
		}
		normalized[workflowID] = applyManifestDefaults(workflowConfig, manifest)
	}
	return normalized, nil
}

func applyManifestDefaults(config map[string]any, manifest workflowmanifest.Manifest) map[string]any {
	if config == nil {
		return nil
	}
	normalized := deepCopyMap(config)
	for _, stage := range manifest.Stages {
		rawStage, ok := normalized[stage.StageID]
		if !ok {
			continue
		}
		stageConfig, ok := rawStage.(map[string]any)
		if !ok {
			continue
		}
		rawTools, ok := stageConfig["tools"]
		if !ok {
			continue
		}
		toolsConfig, ok := rawTools.(map[string]any)
		if !ok {
			continue
		}
		for _, tool := range stage.Tools {
			rawTool, ok := toolsConfig[tool.ToolID]
			if !ok {
				continue
			}
			toolConfig, ok := rawTool.(map[string]any)
			if !ok {
				continue
			}
			enabled, _ := toolConfig["enabled"].(bool)
			if !enabled {
				continue
			}
			for _, param := range tool.Params {
				if _, exists := toolConfig[param.ConfigKey]; exists || param.DefaultValue == nil {
					continue
				}
				toolConfig[param.ConfigKey] = defaultValueForParam(param)
			}
		}
	}
	return normalized
}

func defaultValueForParam(param workflowmanifest.ManifestParam) any {
	switch strings.TrimSpace(param.ValueType) {
	case "integer":
		switch value := param.DefaultValue.(type) {
		case int:
			return value
		case int32:
			return int(value)
		case int64:
			return int(value)
		case float64:
			if math.Trunc(value) == value {
				return int(value)
			}
		}
	case "boolean":
		if value, ok := param.DefaultValue.(bool); ok {
			return value
		}
	case "string":
		if value, ok := param.DefaultValue.(string); ok {
			return value
		}
	}
	return deepCopyAny(param.DefaultValue)
}

func deepCopyMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	copied := make(map[string]any, len(input))
	for key, value := range input {
		copied[key] = deepCopyAny(value)
	}
	return copied
}

func deepCopyAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return deepCopyMap(typed)
	case []any:
		copied := make([]any, len(typed))
		for index := range typed {
			copied[index] = deepCopyAny(typed[index])
		}
		return copied
	default:
		return typed
	}
}
