// Package configuration parses scan workflow dynamic configuration.
package configuration

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Configuration is the canonical structured scan workflow configuration object.
type Configuration map[string]any

// StepConfiguration is the common Workflow Step envelope. EngineConfig belongs
// to the Engine, while Enabled belongs to Workflow orchestration, so the two
// fields are validated as separate branches at this boundary.
type StepConfiguration struct {
	Enabled      bool
	EngineConfig map[string]any
}

// EncodeStepConfigurations emits the only persisted/request shape accepted by
// the Workflow configuration contract. Disabled Steps deliberately discard
// any editor-only Engine draft; enabled Steps retain only their validated
// Engine configuration.
func EncodeStepConfigurations(configs map[string]StepConfiguration) Configuration {
	steps := make(map[string]any, len(configs))
	for stepID, config := range configs {
		entry := map[string]any{"enabled": config.Enabled}
		if config.Enabled {
			entry["engineConfig"] = cloneMap(config.EngineConfig)
		}
		steps[stepID] = entry
	}
	return Configuration{"steps": steps}
}

// ViolationReason is the stable reason vocabulary for Workflow envelope errors.
type ViolationReason string

const (
	ReasonFieldRequired     ViolationReason = "FIELD_REQUIRED"
	ReasonFieldTypeInvalid  ViolationReason = "FIELD_TYPE_INVALID"
	ReasonFieldUnknown      ViolationReason = "FIELD_UNKNOWN"
	ReasonFieldShapeInvalid ViolationReason = "FIELD_SHAPE_INVALID"
	ReasonFieldValueInvalid ViolationReason = "FIELD_VALUE_INVALID"
)

// Violation identifies the first invalid field in a canonical configuration.
// Path is intentionally dynamic because Workflow Step IDs are data.
type Violation struct {
	Path    string
	Reason  ViolationReason
	Message string
}

// ValidationError is returned for every Workflow envelope violation. Callers
// should use Violation rather than parsing Error() text.
type ValidationError struct {
	Violation Violation
}

func (err *ValidationError) Error() string {
	if err == nil {
		return "workflow configuration is invalid"
	}
	if err.Violation.Message == "" {
		return fmt.Sprintf("%s: %s", err.Violation.Path, err.Violation.Reason)
	}
	return fmt.Sprintf("%s: %s", err.Violation.Path, err.Violation.Message)
}

// FirstViolation returns the typed first failure, if err is a Workflow
// configuration validation error.
func FirstViolation(err error) (Violation, bool) {
	if err == nil {
		return Violation{}, false
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) || validationErr == nil {
		return Violation{}, false
	}
	return validationErr.Violation, true
}

func violation(path string, reason ViolationReason, message string) error {
	return &ValidationError{Violation: Violation{Path: path, Reason: reason, Message: message}}
}

// Decode validates a submitted canonical Workflow configuration. It requires
// every known Step, an explicit boolean discriminator, strict conditional
// branches, and at least one enabled Step. No Profile or metadata lookup is
// performed here.
func Decode(configuration any, knownSteps map[string]struct{}) (map[string]StepConfiguration, error) {
	return decode(configuration, knownSteps, true)
}

// ValidateAtLeastOneEnabled applies the request-only cross-Step rule to a
// previously decoded configuration. Profile drafts intentionally do not use
// this rule because an all-disabled draft is a valid editing state.
func ValidateAtLeastOneEnabled(configs map[string]StepConfiguration) error {
	for _, config := range configs {
		if config.Enabled {
			return nil
		}
	}
	return violation("configuration.steps", ReasonFieldValueInvalid, "at least one workflow Step must be enabled")
}

// ExtractStepConfigurations validates the exact Step envelope but leaves the
// at-least-one rule to the caller. This is retained for Profile/read-side
// consumers and older package callers; submitted request paths use Decode.
func ExtractStepConfigurations(configuration any, knownSteps map[string]struct{}) (map[string]StepConfiguration, error) {
	return decode(configuration, knownSteps, false)
}

// ExtractCompleteStepConfigurations is the descriptive entry point used by
// older Scan and Scheduled Scan boundaries. New request paths should call
// Decode so the shared at-least-one rule is applied by the decoder itself.
func ExtractCompleteStepConfigurations(configuration any, knownSteps map[string]struct{}) (map[string]StepConfiguration, error) {
	return ExtractStepConfigurations(configuration, knownSteps)
}

func decode(configuration any, knownSteps map[string]struct{}, requireEnabled bool) (map[string]StepConfiguration, error) {
	root, err := normalizeRoot(configuration)
	if err != nil {
		return nil, err
	}
	if len(root) == 0 {
		return nil, violation("configuration.steps", ReasonFieldRequired, "configuration must contain only top-level field steps; steps is required")
	}
	rootKeys := sortedKeys(root)
	for _, key := range rootKeys {
		if key != "steps" {
			return nil, violation("configuration."+key, ReasonFieldUnknown, "configuration must contain only top-level field steps; unknown top-level field")
		}
	}
	rawSteps, ok := root["steps"]
	if !ok {
		return nil, violation("configuration.steps", ReasonFieldRequired, "steps is required")
	}
	steps, ok := objectMap(rawSteps)
	if !ok {
		return nil, violation("configuration.steps", ReasonFieldTypeInvalid, "configuration.steps must be object")
	}

	stepIDs := sortedStringSet(knownSteps)
	providedIDs := sortedKeys(steps)
	for _, stepID := range providedIDs {
		if _, known := knownSteps[stepID]; !known {
			return nil, violation(stepPath(stepID), ReasonFieldUnknown, "unknown workflow step")
		}
	}
	for _, stepID := range stepIDs {
		if _, provided := steps[stepID]; !provided {
			return nil, violation("configuration.steps", ReasonFieldShapeInvalid, "configuration.steps must contain exactly every workflow step; missing "+stepID)
		}
	}

	configs := make(map[string]StepConfiguration, len(stepIDs))
	for _, stepID := range stepIDs {
		path := stepPath(stepID)
		rawEntry := steps[stepID]
		entry, ok := objectMap(rawEntry)
		if !ok {
			return nil, violation(path, ReasonFieldTypeInvalid, "configuration.steps."+stepID+" must be object")
		}
		rawEnabled, exists := entry["enabled"]
		if !exists {
			return nil, violation(path+".enabled", ReasonFieldRequired, "enabled is required")
		}
		enabled, ok := rawEnabled.(bool)
		if !ok {
			return nil, violation(path+".enabled", ReasonFieldTypeInvalid, "enabled must be boolean")
		}
		if enabled {
			rawEngineConfig, exists := entry["engineConfig"]
			if !exists {
				return nil, violation(path+".engineConfig", ReasonFieldRequired, "engineConfig is required when enabled")
			}
			if unknown := firstUnknownEntryKey(entry, "enabled", "engineConfig"); unknown != "" {
				return nil, violation(path+"."+unknown, ReasonFieldUnknown, "enabled branch requires only enabled and engineConfig; unknown Step field")
			}
			engineConfig, ok := objectMap(rawEngineConfig)
			if !ok {
				return nil, violation(path+".engineConfig", ReasonFieldTypeInvalid, "configuration.steps."+stepID+".engineConfig must be object")
			}
			configs[stepID] = StepConfiguration{Enabled: true, EngineConfig: cloneMap(engineConfig)}
			continue
		}
		if len(entry) != 1 {
			if _, hasEngineConfig := entry["engineConfig"]; hasEngineConfig {
				return nil, violation(path+".engineConfig", ReasonFieldShapeInvalid, "disabled branch only supports enabled; disabled Step must not contain engineConfig")
			}
			if unknown := firstUnknownEntryKey(entry, "enabled"); unknown != "" {
				return nil, violation(path+"."+unknown, ReasonFieldUnknown, "unknown Step field")
			}
			return nil, violation(path, ReasonFieldShapeInvalid, "disabled Step only supports enabled")
		}
		configs[stepID] = StepConfiguration{Enabled: false}
	}
	if requireEnabled {
		if err := ValidateAtLeastOneEnabled(configs); err != nil {
			return nil, err
		}
	}
	return configs, nil
}

func stepPath(stepID string) string {
	return `configuration.steps["` + stepID + `"]`
}

func firstUnknownEntryKey(entry map[string]any, allowed ...string) string {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowedSet[key] = struct{}{}
	}
	for _, key := range sortedKeys(entry) {
		if _, ok := allowedSet[key]; !ok {
			return key
		}
	}
	return ""
}

// Normalize returns configuration as map[string]any and recursively normalizes
// nested YAML maps. It remains a general conversion helper; Decode performs
// the strict contract validation.
func Normalize(configuration any) (Configuration, error) {
	normalized, err := normalizeValue(configuration, "configuration")
	if err != nil {
		return nil, err
	}
	root, ok := normalized.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("configuration must be an object")
	}
	return Configuration(root), nil
}

func normalizeRoot(configuration any) (map[string]any, error) {
	normalized, err := normalizeValue(configuration, "configuration")
	if err != nil {
		return nil, err
	}
	root, ok := normalized.(map[string]any)
	if !ok {
		return nil, violation("configuration", ReasonFieldTypeInvalid, "configuration must be an object")
	}
	return root, nil
}

func objectMap(value any) (map[string]any, bool) {
	object, ok := value.(map[string]any)
	return object, ok
}

func normalizeValue(value any, path string) (any, error) {
	switch typed := value.(type) {
	case Configuration:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			normalized, err := normalizeValue(nested, path+"."+key)
			if err != nil {
				return nil, err
			}
			out[key] = normalized
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			normalized, err := normalizeValue(nested, path+"."+key)
			if err != nil {
				return nil, err
			}
			out[key] = normalized
		}
		return out, nil
	case map[any]any:
		out := make(map[string]any, len(typed))
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keyText, ok := key.(string)
			if !ok || strings.TrimSpace(keyText) == "" {
				return nil, violation(path, ReasonFieldTypeInvalid, "object keys must be non-empty strings")
			}
			keys = append(keys, keyText)
		}
		sort.Strings(keys)
		for _, keyText := range keys {
			normalized, err := normalizeValue(typed[keyText], path+"."+keyText)
			if err != nil {
				return nil, err
			}
			out[keyText] = normalized
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for index, item := range typed {
			normalized, err := normalizeValue(item, fmt.Sprintf("%s[%d]", path, index))
			if err != nil {
				return nil, err
			}
			out[index] = normalized
		}
		return out, nil
	case nil:
		return nil, nil
	default:
		return value, nil
	}
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		switch nested := value.(type) {
		case map[string]any:
			out[key] = cloneMap(nested)
		case Configuration:
			out[key] = cloneMap(map[string]any(nested))
		case []any:
			items := make([]any, len(nested))
			for index, item := range nested {
				if object, ok := item.(map[string]any); ok {
					items[index] = cloneMap(object)
				} else {
					items[index] = item
				}
			}
			out[key] = items
		default:
			out[key] = value
		}
	}
	return out
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedStringSet(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
