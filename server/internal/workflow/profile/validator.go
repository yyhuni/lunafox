package profile

import (
	"fmt"
	"strings"

	workflowschema "github.com/yyhuni/lunafox/server/internal/workflow/schema"
	"gopkg.in/yaml.v3"
)

// ValidateConfiguration validates the configuration YAML against all applicable schemas.
// It rejects unknown workflow keys and validates each known workflow section against its schema.
func ValidateConfiguration(configuration string) error {
	if configuration == "" {
		return nil
	}

	workflowIDs, err := ExtractWorkflowIDs(configuration)
	if err != nil {
		return err
	}

	for _, workflow := range workflowIDs {
		if err := workflowschema.ValidateYAML(workflow, []byte(configuration)); err != nil {
			return fmt.Errorf("workflow %s: %w", workflow, err)
		}
	}

	return nil
}

// ExtractWorkflowIDs returns top-level workflow keys in declaration order.
// Unknown keys are rejected to keep profile contracts strict.
func ExtractWorkflowIDs(configuration string) ([]string, error) {
	trimmed := strings.TrimSpace(configuration)
	if trimmed == "" {
		return []string{}, nil
	}

	keys, err := parseTopLevelKeys(trimmed)
	if err != nil {
		return nil, err
	}

	known, err := knownWorkflowSet()
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}

	workflows := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, exists := known[key]; !exists {
			return nil, fmt.Errorf("unknown workflow key %q", key)
		}
		workflows = append(workflows, key)
	}
	return workflows, nil
}

func parseTopLevelKeys(configuration string) ([]string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(configuration), &root); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		return nil, fmt.Errorf("configuration YAML must be a mapping")
	}
	mapping := root.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("configuration YAML must be a mapping")
	}

	seen := make(map[string]struct{}, len(mapping.Content)/2)
	keys := make([]string, 0, len(mapping.Content)/2)
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		key := strings.TrimSpace(mapping.Content[index].Value)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys, nil
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
