package workflowdefaulting

import (
	"fmt"
	"strings"

	workflowmanifest "github.com/yyhuni/lunafox/server/internal/workflow/manifest"
	workflowprofile "github.com/yyhuni/lunafox/server/internal/workflow/profile"
)

func NormalizeRootConfig(root map[string]any, workflowIDs []string) (map[string]any, error) {
	if root == nil {
		return nil, fmt.Errorf("configuration root must be object")
	}
	profileLoader, err := workflowprofile.NewLoader()
	if err != nil {
		return nil, fmt.Errorf("load workflow profiles: %w", err)
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
		defaultProfile := profileLoader.GetByID(workflowID)
		if defaultProfile == nil {
			return nil, fmt.Errorf("default profile %q not found for workflow %q", workflowID, workflowID)
		}
		rawWorkflowConfig, ok := normalized[workflowID]
		if !ok {
			return nil, fmt.Errorf("missing %s config; expected nested configuration under key %q", workflowID, workflowID)
		}
		workflowConfig, ok := rawWorkflowConfig.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("workflow %s configuration must be object", workflowID)
		}
		defaultRootConfig := defaultProfile.Configuration
		defaultWorkflowConfig, ok := asStringAnyMap(defaultRootConfig[workflowID])
		if !ok {
			return nil, fmt.Errorf("default profile %q missing workflow config for %q", workflowID, workflowID)
		}
		normalized[workflowID] = applyDefaultProfileConfig(workflowConfig, defaultWorkflowConfig, manifest)
	}
	return normalized, nil
}

func applyDefaultProfileConfig(config map[string]any, defaultWorkflowConfig map[string]any, manifest workflowmanifest.Manifest) map[string]any {
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
		defaultStageConfig, _ := asStringAnyMap(defaultWorkflowConfig[stage.StageID])
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
			defaultTools, _ := asStringAnyMap(defaultStageConfig["tools"])
			defaultToolConfig, _ := asStringAnyMap(defaultTools[tool.ToolID])
			enabled, _ := toolConfig["enabled"].(bool)
			if !enabled {
				continue
			}
			for defaultKey, defaultValue := range defaultToolConfig {
				if defaultKey == "enabled" {
					continue
				}
				if _, exists := toolConfig[defaultKey]; exists {
					continue
				}
				toolConfig[defaultKey] = deepCopyAny(defaultValue)
			}
		}
	}
	return normalized
}

func asStringAnyMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case workflowprofile.WorkflowConfig:
		return map[string]any(typed), true
	default:
		return nil, false
	}
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
