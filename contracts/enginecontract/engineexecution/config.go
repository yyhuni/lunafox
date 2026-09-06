package engineexecution

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// NormalizeAndValidateConfig applies engine.v5 parameter defaults and
// validates the complete closed task-config shape without mutating its input.
// Missing sections and enabled flags are completed only from the selected
// package definition, so Server planning has one authoritative default source.
func NormalizeAndValidateConfig(raw map[string]any, definition ExecutionDefinition) (map[string]any, error) {
	normalizedDefinition, err := NormalizeExecutionDefinition(definition)
	if err != nil {
		return nil, fmt.Errorf("invalid execution definition: %w", err)
	}
	if raw == nil {
		return nil, fmt.Errorf("config is required")
	}

	normalized := deepCopyMap(raw)
	sections := make(map[string]ConfigSectionDefinition, len(normalizedDefinition.ConfigSections))
	for _, section := range normalizedDefinition.ConfigSections {
		sections[section.ID] = section
	}
	for _, sectionID := range sortedMapKeys(normalized) {
		if _, exists := sections[sectionID]; !exists {
			return nil, fmt.Errorf("unknown config section %q", sectionID)
		}
	}

	for _, section := range normalizedDefinition.ConfigSections {
		rawSection, exists := normalized[section.ID]
		if !exists {
			rawSection = map[string]any{}
			normalized[section.ID] = rawSection
		}
		sectionConfig, ok := rawSection.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s must be object", section.ID)
		}

		params := make(map[string]ParamDefinition, len(section.Params))
		for _, param := range section.Params {
			params[param.Key] = param
		}
		enabledValue, exists := sectionConfig["enabled"]
		if !exists {
			enabledValue = section.DefaultEnabled
			sectionConfig["enabled"] = enabledValue
		}
		enabled, ok := enabledValue.(bool)
		if !ok {
			return nil, fmt.Errorf("%s.enabled must be boolean", section.ID)
		}
		if !enabled {
			if section.RequiredEnabled {
				return nil, fmt.Errorf("%s requiredEnabled config section must be enabled", section.ID)
			}
			for _, key := range sortedMapKeys(sectionConfig) {
				if key != "enabled" {
					return nil, fmt.Errorf("%s disabled config section must contain only enabled", section.ID)
				}
			}
			normalized[section.ID] = map[string]any{"enabled": false}
			continue
		}
		for _, key := range sortedMapKeys(sectionConfig) {
			if key == "enabled" {
				continue
			}
			if _, exists := params[key]; !exists {
				return nil, fmt.Errorf("%s.%s is not allowed", section.ID, key)
			}
		}

		for _, param := range section.Params {
			value, exists := sectionConfig[param.Key]
			if !exists {
				if param.Default != nil {
					value = deepCopyAny(param.Default)
					exists = true
				} else if enabled {
					return nil, fmt.Errorf("%s.%s is required when enabled", section.ID, param.Key)
				}
			}
			if !exists {
				continue
			}

			path := section.ID + "." + param.Key
			normalizedValue, err := normalizeRuntimeParamValue(path, value, param)
			if err != nil {
				return nil, err
			}
			sectionConfig[param.Key] = normalizedValue
		}
	}
	if !hasEnabledConfigSection(normalized, normalizedDefinition.ConfigSections) {
		return nil, fmt.Errorf("at least one config section must be enabled")
	}

	return normalized, nil
}

// ValidateCompleteConfig validates an already materialized configuration. It
// never supplies defaults or deep-merges caller input, so the value returned is
// exactly the complete configuration a Scan will freeze into its saved plan.
func ValidateCompleteConfig(raw map[string]any, definition ExecutionDefinition) (map[string]any, error) {
	normalizedDefinition, err := NormalizeExecutionDefinition(definition)
	if err != nil {
		return nil, fmt.Errorf("invalid execution definition: %w", err)
	}
	if raw == nil {
		return nil, fmt.Errorf("config is required")
	}
	complete := deepCopyMap(raw)
	sections := make(map[string]ConfigSectionDefinition, len(normalizedDefinition.ConfigSections))
	for _, section := range normalizedDefinition.ConfigSections {
		sections[section.ID] = section
	}
	for _, sectionID := range sortedMapKeys(complete) {
		if _, exists := sections[sectionID]; !exists {
			return nil, fmt.Errorf("unknown config section %q", sectionID)
		}
	}
	for _, section := range normalizedDefinition.ConfigSections {
		rawSection, exists := complete[section.ID]
		if !exists {
			return nil, fmt.Errorf("%s is required", section.ID)
		}
		sectionConfig, ok := rawSection.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s must be object", section.ID)
		}
		params := make(map[string]ParamDefinition, len(section.Params))
		for _, param := range section.Params {
			params[param.Key] = param
		}
		enabledValue, exists := sectionConfig["enabled"]
		if !exists {
			return nil, fmt.Errorf("%s.enabled is required", section.ID)
		}
		enabled, ok := enabledValue.(bool)
		if !ok {
			return nil, fmt.Errorf("%s.enabled must be boolean", section.ID)
		}
		if !enabled {
			if section.RequiredEnabled {
				return nil, fmt.Errorf("%s requiredEnabled config section must be enabled", section.ID)
			}
			if len(sectionConfig) != 1 {
				return nil, fmt.Errorf("%s disabled config section must contain only enabled", section.ID)
			}
			continue
		}
		for _, key := range sortedMapKeys(sectionConfig) {
			if key == "enabled" {
				continue
			}
			if _, exists := params[key]; !exists {
				return nil, fmt.Errorf("%s.%s is not allowed", section.ID, key)
			}
		}
		for _, param := range section.Params {
			value, exists := sectionConfig[param.Key]
			if !exists {
				return nil, fmt.Errorf("%s.%s is required", section.ID, param.Key)
			}
			normalizedValue, err := normalizeRuntimeParamValue(section.ID+"."+param.Key, value, param)
			if err != nil {
				return nil, err
			}
			sectionConfig[param.Key] = normalizedValue
		}
	}
	if !hasEnabledConfigSection(complete, normalizedDefinition.ConfigSections) {
		return nil, fmt.Errorf("at least one config section must be enabled")
	}
	return complete, nil
}

func hasEnabledConfigSection(config map[string]any, sections []ConfigSectionDefinition) bool {
	for _, section := range sections {
		sectionConfig, ok := config[section.ID].(map[string]any)
		if !ok {
			continue
		}
		enabled, ok := sectionConfig["enabled"].(bool)
		if ok && enabled {
			return true
		}
	}
	return false
}

func normalizeRuntimeParamValue(path string, value any, param ParamDefinition) (any, error) {
	switch param.Type {
	case ParamTypeInteger:
		integer, ok := integerValue(value)
		if !ok {
			return nil, fmt.Errorf("%s must be integer", path)
		}
		if param.Minimum != nil && integer < int64(*param.Minimum) {
			return nil, fmt.Errorf("%s must be >= %d", path, *param.Minimum)
		}
		if param.Maximum != nil && integer > int64(*param.Maximum) {
			return nil, fmt.Errorf("%s must be <= %d", path, *param.Maximum)
		}
		return canonicalInteger(integer), nil
	case ParamTypeBoolean:
		boolean, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("%s must be boolean", path)
		}
		return boolean, nil
	case ParamTypeString:
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s must be string", path)
		}
		// Resource defaults apply only when the value is absent. An explicit
		// empty selection is invalid and must never be repaired with a fallback.
		if param.Resource != nil && strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("%s must be non-empty", path)
		}
		if param.Enum != nil && !stringInEnum(text, param.Enum) {
			return nil, fmt.Errorf("%s must be one of: %s", path, strings.Join(param.Enum, ", "))
		}
		if param.MinLength != nil && utf8.RuneCountInString(text) < *param.MinLength {
			return nil, fmt.Errorf("%s length must be >= %d", path, *param.MinLength)
		}
		if param.MaxLength != nil && utf8.RuneCountInString(text) > *param.MaxLength {
			return nil, fmt.Errorf("%s length must be <= %d", path, *param.MaxLength)
		}
		if param.Pattern != "" {
			pattern, err := regexp.Compile(param.Pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid execution definition: %s has invalid pattern: %w", path, err)
			}
			if !pattern.MatchString(text) {
				return nil, fmt.Errorf("%s must match pattern %q", path, param.Pattern)
			}
		}
		return text, nil
	case ParamTypeStringArray:
		values, ok := stringArrayValue(value)
		if !ok {
			return nil, fmt.Errorf("%s must be string array", path)
		}
		if param.MinItems != nil && len(values) < *param.MinItems {
			return nil, fmt.Errorf("%s must contain at least %d items", path, *param.MinItems)
		}
		if param.MaxItems != nil && len(values) > *param.MaxItems {
			return nil, fmt.Errorf("%s must contain at most %d items", path, *param.MaxItems)
		}
		if param.MinItems != nil || param.MaxItems != nil {
			seen := make(map[string]struct{}, len(values))
			for _, text := range values {
				if _, exists := seen[text]; exists {
					return nil, fmt.Errorf("%s must not contain duplicate values", path)
				}
				seen[text] = struct{}{}
			}
		}
		if (param.MinItems != nil || param.MaxItems != nil) && param.Enum != nil {
			order := make(map[string]int, len(param.Enum))
			for index, candidate := range param.Enum {
				order[candidate] = index
			}
			sort.SliceStable(values, func(left, right int) bool { return order[values[left]] < order[values[right]] })
		}
		for _, text := range values {
			if param.Enum != nil && !stringInEnum(text, param.Enum) {
				return nil, fmt.Errorf("%s contains a value not in: %s", path, strings.Join(param.Enum, ", "))
			}
		}
		return values, nil
	default:
		return nil, fmt.Errorf("invalid execution definition: %s has unsupported type %q", path, param.Type)
	}
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func canonicalInteger(value int64) any {
	if value >= int64(minInt()) && value <= int64(maxInt()) {
		return int(value)
	}
	return value
}
