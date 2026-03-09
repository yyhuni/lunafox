package profile

import (
	"fmt"
	"sort"
	"strings"

	workflowschema "github.com/yyhuni/lunafox/server/internal/workflow/schema"
)

// ValidateConfiguration validates the configuration object against all applicable schemas.
// It rejects unknown workflow keys and validates each known workflow section against its schema.
func ValidateConfiguration(configuration any) error {
	root, err := normalizeConfiguration(configuration)
	if err != nil {
		return err
	}
	if len(root) == 0 {
		return nil
	}

	workflowIDs, err := ValidateAndExtractWorkflowIDs(root)
	if err != nil {
		return err
	}

	for _, workflowID := range workflowIDs {
		workflowConfig, ok := asConfigMap(root[workflowID])
		if !ok {
			return fmt.Errorf("workflow %q config must be a mapping", workflowID)
		}
		if err := workflowschema.ValidateConfigMap(workflowID, workflowConfig); err != nil {
			return fmt.Errorf("workflow %s: %w", workflowID, err)
		}
	}

	return nil
}

// ExtractWorkflowIDs returns top-level workflow keys in stable order.
// It only extracts IDs and does not validate whether they are known workflows.
func ExtractWorkflowIDs(configuration any) ([]string, error) {
	root, err := normalizeConfiguration(configuration)
	if err != nil {
		return nil, err
	}
	return extractWorkflowIDsFromConfig(root), nil
}

// ValidateAndExtractWorkflowIDs returns top-level workflow keys in stable order.
// Unknown keys are rejected to keep profile contracts strict.
func ValidateAndExtractWorkflowIDs(configuration any) ([]string, error) {
	root, err := normalizeConfiguration(configuration)
	if err != nil {
		return nil, err
	}

	workflowIDs := extractWorkflowIDsFromConfig(root)
	if len(workflowIDs) == 0 {
		return workflowIDs, nil
	}

	known, err := knownWorkflowSet()
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}

	for _, workflowID := range workflowIDs {
		if _, exists := known[workflowID]; !exists {
			return nil, fmt.Errorf("unknown workflow key %q", workflowID)
		}
	}

	return workflowIDs, nil
}

func extractWorkflowIDsFromConfig(root WorkflowConfig) []string {
	if len(root) == 0 {
		return []string{}
	}

	workflowIDs := make([]string, 0, len(root))
	for key := range root {
		workflowID := strings.TrimSpace(key)
		if workflowID == "" {
			continue
		}
		workflowIDs = append(workflowIDs, workflowID)
	}

	sort.Strings(workflowIDs)
	return workflowIDs
}

func normalizeConfiguration(configuration any) (WorkflowConfig, error) {
	switch value := configuration.(type) {
	case nil:
		return nil, nil
	case WorkflowConfig:
		return value, nil
	case map[string]any:
		return WorkflowConfig(value), nil
	default:
		return nil, fmt.Errorf("configuration must be an object")
	}
}

func asConfigMap(value any) (map[string]any, bool) {
	switch config := value.(type) {
	case nil:
		return nil, false
	case map[string]any:
		return config, true
	case WorkflowConfig:
		return map[string]any(config), true
	default:
		return nil, false
	}
}

func knownWorkflowSet() (map[string]struct{}, error) {
	knownWorkflows, err := workflowschema.ListWorkflows()
	if err != nil {
		return nil, err
	}

	set := make(map[string]struct{}, len(knownWorkflows))
	for _, workflow := range knownWorkflows {
		name := strings.TrimSpace(workflow)
		if name == "" {
			continue
		}
		set[name] = struct{}{}
	}
	return set, nil
}
