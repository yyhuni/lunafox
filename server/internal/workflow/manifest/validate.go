package workflowmanifest

import (
	"fmt"
	"strings"

	workflowprofile "github.com/yyhuni/lunafox/server/internal/workflow/profile"
)

func validateManifest(manifest Manifest, knownProfileIDs map[string]struct{}) error {
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

	expectedSchemaID := fmt.Sprintf("lunafox://schemas/workflows/%s", workflowID)
	if strings.TrimSpace(manifest.ConfigSchemaID) != expectedSchemaID {
		return fmt.Errorf("configSchemaId must be %q", expectedSchemaID)
	}
	if len(manifest.SupportedTargetTypeIDs) == 0 {
		return fmt.Errorf("supportedTargetTypeIds must not be empty")
	}

	targetTypes := map[string]struct{}{}
	for _, targetTypeID := range manifest.SupportedTargetTypeIDs {
		targetTypeID = strings.TrimSpace(targetTypeID)
		if targetTypeID == "" {
			return fmt.Errorf("supportedTargetTypeIds must not contain empty values")
		}
		if _, exists := targetTypes[targetTypeID]; exists {
			return fmt.Errorf("duplicate supportedTargetTypeId %q", targetTypeID)
		}
		targetTypes[targetTypeID] = struct{}{}
	}

	defaultProfileID := strings.TrimSpace(manifest.DefaultProfileID)
	if defaultProfileID == "" {
		return fmt.Errorf("defaultProfileId is required")
	}
	if _, exists := knownProfileIDs[defaultProfileID]; !exists {
		return fmt.Errorf("defaultProfileId %q not found", defaultProfileID)
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

func loadKnownProfileIDs() (map[string]struct{}, error) {
	loader, err := workflowprofile.NewLoader()
	if err != nil {
		return nil, fmt.Errorf("load workflow profiles: %w", err)
	}

	known := make(map[string]struct{}, len(loader.List()))
	for _, profile := range loader.List() {
		profileID := strings.TrimSpace(profile.ID)
		if profileID == "" {
			continue
		}
		known[profileID] = struct{}{}
	}

	return known, nil
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
