package workflowmanifest

import (
	"fmt"
	"strings"
)

func validateManifest(manifest Manifest) error {
	workflowID := strings.TrimSpace(manifest.WorkflowID)
	if workflowID == "" {
		return fmt.Errorf("workflowId is required")
	}
	if err := validateWorkflowID(workflowID); err != nil {
		return err
	}
	if strings.TrimSpace(manifest.ManifestVersion) == "" {
		return fmt.Errorf("manifestVersion is required")
	}
	if strings.TrimSpace(manifest.DisplayName) == "" {
		return fmt.Errorf("displayName is required")
	}
	if strings.TrimSpace(manifest.Executor.Type) == "" {
		return fmt.Errorf("executor.type is required")
	}
	if strings.TrimSpace(manifest.Executor.Ref) == "" {
		return fmt.Errorf("executor.ref is required")
	}
	switch strings.TrimSpace(manifest.Executor.Type) {
	case "builtin", "plugin":
	default:
		return fmt.Errorf("unsupported executor.type %q", manifest.Executor.Type)
	}

	if len(manifest.Stages) == 0 {
		return fmt.Errorf("stages must not be empty")
	}

	stageSet := map[string]struct{}{}
	for _, stage := range manifest.Stages {
		stageID := strings.TrimSpace(stage.StageID)
		if stageID == "" {
			return fmt.Errorf("stageId is required")
		}
		if !componentIDPattern.MatchString(stageID) {
			return fmt.Errorf("invalid stageId %q", stageID)
		}
		if _, exists := stageSet[stageID]; exists {
			return fmt.Errorf("duplicate stageId %q", stageID)
		}
		stageSet[stageID] = struct{}{}
		if strings.TrimSpace(stage.DisplayName) == "" {
			return fmt.Errorf("stage %s displayName is required", stageID)
		}
		if len(stage.Tools) == 0 {
			return fmt.Errorf("stage %s must define at least one tool", stageID)
		}

		toolSet := map[string]struct{}{}
		for _, tool := range stage.Tools {
			toolID := strings.TrimSpace(tool.ToolID)
			if toolID == "" {
				return fmt.Errorf("toolId is required in stage %s", stageID)
			}
			if !componentIDPattern.MatchString(toolID) {
				return fmt.Errorf("invalid toolId %q in stage %s", toolID, stageID)
			}
			if _, exists := toolSet[toolID]; exists {
				return fmt.Errorf("duplicate toolId %q in stage %s", toolID, stageID)
			}
			toolSet[toolID] = struct{}{}

			paramSet := map[string]struct{}{}
			for _, param := range tool.Params {
				configKey := strings.TrimSpace(param.ConfigKey)
				if configKey == "" {
					return fmt.Errorf("configKey is required in tool %s", toolID)
				}
				if !configKeyPattern.MatchString(configKey) {
					return fmt.Errorf("invalid configKey %q in tool %s", configKey, toolID)
				}
				if _, exists := paramSet[configKey]; exists {
					return fmt.Errorf("duplicate configKey %q in tool %s", configKey, toolID)
				}
				paramSet[configKey] = struct{}{}

				switch strings.TrimSpace(param.ValueType) {
				case "integer", "boolean", "string":
				default:
					return fmt.Errorf("invalid valueType %q for %s.%s", param.ValueType, toolID, configKey)
				}
			}
		}
	}

	return nil
}

func validateManifestList(items []Manifest) error {
	seen := map[string]struct{}{}
	for _, item := range items {
		workflowID := strings.TrimSpace(item.WorkflowID)
		if workflowID == "" {
			return fmt.Errorf("workflowId must not be empty")
		}
		if _, exists := seen[workflowID]; exists {
			return fmt.Errorf("duplicate manifest for workflow %q", workflowID)
		}
		seen[workflowID] = struct{}{}
	}
	return nil
}

func validateWorkflowID(workflowID string) error {
	if !workflowIDPattern.MatchString(workflowID) {
		return fmt.Errorf("invalid workflowId %q", workflowID)
	}
	if _, reserved := reservedWorkflowIDNames[workflowID]; reserved {
		return fmt.Errorf("reserved workflowId %q is not allowed", workflowID)
	}
	return nil
}
