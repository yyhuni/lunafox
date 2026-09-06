package engineexecution

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	TargetTypeDomain = "domain"
	TargetTypeIP     = "ip"
	TargetTypeCIDR   = "cidr"

	InputSubdomains   = "subdomains"
	InputHostPorts    = "hostPorts"
	InputWebsiteURLs  = "websiteURLs"
	InputEndpointURLs = "endpointURLs"

	PlatformResourceSubfinderProviderConfig          = "subfinderProviderConfig"
	PlatformResourceFingerprintLibraryFingerPrintHub = "fingerprintLibraryFingerPrintHub"
	NucleiTemplates                                  = "nucleiTemplates"

	ParamTypeBoolean     = "boolean"
	ParamTypeInteger     = "integer"
	ParamTypeString      = "string"
	ParamTypeStringArray = "stringArray"

	ConfigResourceKindWordlist = "wordlist"
)

var (
	canonicalConfigSectionID = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)
	canonicalConfigParamKey  = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
)

// ExecutionDefinition is the normalized execution declaration derived from
// a strict engine.v5 manifest. It is an authoring contract consumed by
// package validation, Server planning, catalog projection, and generators.
type ExecutionDefinition struct {
	EngineAPIMajor       uint32                    `json:"engineApiMajor"`
	SupportedTargetTypes []string                  `json:"supportedTargetTypes"`
	ConfigSections       []ConfigSectionDefinition `json:"configSections"`
	ExecutionResources   []string                  `json:"executionResources,omitempty"`
}

type ConfigSectionDefinition struct {
	ID             string `json:"id"`
	DefaultEnabled bool   `json:"defaultEnabled,omitempty"`
	// RequiredEnabled is limited to one unconditional section invariant. It
	// cannot encode a group, condition, or Engine runtime orchestration rule.
	RequiredEnabled bool              `json:"requiredEnabled,omitempty"`
	Params          []ParamDefinition `json:"params,omitempty"`
}

type ParamDefinition struct {
	Key       string                `json:"key"`
	Type      string                `json:"type,omitempty"`
	Default   any                   `json:"default,omitempty"`
	Minimum   *int                  `json:"minimum,omitempty"`
	Maximum   *int                  `json:"maximum,omitempty"`
	MinLength *int                  `json:"minLength,omitempty"`
	MaxLength *int                  `json:"maxLength,omitempty"`
	MinItems  *int                  `json:"minItems,omitempty"`
	MaxItems  *int                  `json:"maxItems,omitempty"`
	Pattern   string                `json:"pattern,omitempty"`
	Enum      []string              `json:"enum,omitempty"`
	Resource  *ParamResourceBinding `json:"resource,omitempty"`
}

type ParamResourceBinding struct {
	Kind string `json:"kind"`
}

// MarshalJSON preserves the distinction between an omitted execution-resource
// declaration and an explicitly present empty set.
func (definition ExecutionDefinition) MarshalJSON() ([]byte, error) {
	type wireDefinition ExecutionDefinition
	if definition.ExecutionResources == nil {
		return json.Marshal(wireDefinition(definition))
	}
	return json.Marshal(struct {
		wireDefinition
		ExecutionResources []string `json:"executionResources"`
	}{
		wireDefinition:     wireDefinition(definition),
		ExecutionResources: definition.ExecutionResources,
	})
}

// ValidateExecutionDefinition validates the normalized engine.v5 execution
// declaration before any downstream consumer observes it.
func ValidateExecutionDefinition(definition ExecutionDefinition) error {
	if definition.EngineAPIMajor == 0 {
		return fmt.Errorf("execution.engineApiMajor must be non-zero")
	}
	if err := validateClosedStringSet(
		"execution.supportedTargetTypes",
		definition.SupportedTargetTypes,
		false,
		[]string{TargetTypeDomain, TargetTypeIP, TargetTypeCIDR},
		"supported target type",
	); err != nil {
		return err
	}
	if err := validateClosedStringSet(
		"execution.executionResources",
		definition.ExecutionResources,
		true,
		[]string{PlatformResourceSubfinderProviderConfig, PlatformResourceFingerprintLibraryFingerPrintHub, NucleiTemplates},
		"execution resource",
	); err != nil {
		// executionResources is optional in engine.v5; a present array remains
		// closed and duplicate-free, while omission carries no declaration.
		if definition.ExecutionResources != nil {
			return err
		}
	}
	if len(definition.ConfigSections) == 0 {
		return fmt.Errorf("execution.configSections must define at least one section")
	}
	sectionIDs := make(map[string]struct{}, len(definition.ConfigSections))
	for _, section := range definition.ConfigSections {
		if err := ValidateConfigSectionID(section.ID); err != nil {
			return err
		}
		if _, exists := sectionIDs[section.ID]; exists {
			return fmt.Errorf("duplicate config section id %q", section.ID)
		}
		sectionIDs[section.ID] = struct{}{}
		if section.RequiredEnabled && !section.DefaultEnabled {
			return fmt.Errorf("config section %q requiredEnabled requires defaultEnabled:true", section.ID)
		}
		if len(section.Params) == 0 {
			return fmt.Errorf("config section %q must define at least one param", section.ID)
		}
		paramKeys := make(map[string]struct{}, len(section.Params))
		for _, param := range section.Params {
			if err := ValidateConfigParamKey(param.Key); err != nil {
				return fmt.Errorf("config section %q: %w", section.ID, err)
			}
			if _, exists := paramKeys[param.Key]; exists {
				return fmt.Errorf("duplicate param key %q in config section %q", param.Key, section.ID)
			}
			paramKeys[param.Key] = struct{}{}
			if err := validateParamDefinition(section.ID, param); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateConfigSectionID validates the canonical lower snake_case identity
// shared by engine.v5 authoring and compiled plan config/resource bindings.
func ValidateConfigSectionID(value string) error {
	if value != strings.TrimSpace(value) || !canonicalConfigSectionID.MatchString(value) {
		return fmt.Errorf("config section id %q must be canonical lower snake_case", value)
	}
	return nil
}

// ValidateConfigParamKey validates the canonical lower kebab-case identity
// shared by engine.v5 authoring and compiled plan config/resource bindings.
func ValidateConfigParamKey(value string) error {
	if value != strings.TrimSpace(value) || !canonicalConfigParamKey.MatchString(value) || value == "enabled" {
		return fmt.Errorf("config param key %q must be canonical lower kebab-case and must not be enabled", value)
	}
	return nil
}

// DecodeExecutionDefinition strictly decodes an engine.v5 execution object
// and returns its detached canonical representation.
func DecodeExecutionDefinition(payload []byte, source string) (ExecutionDefinition, error) {
	if err := ValidateStrictJSONFields(payload); err != nil {
		return ExecutionDefinition{}, fmt.Errorf("decode %q: %w", source, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()

	var definition ExecutionDefinition
	if err := decoder.Decode(&definition); err != nil {
		return ExecutionDefinition{}, fmt.Errorf("decode %q: %w", source, err)
	}
	if err := consumeExecutionDefinitionJSONEOF(decoder); err != nil {
		return ExecutionDefinition{}, fmt.Errorf("decode %q: %w", source, err)
	}
	normalized, err := NormalizeExecutionDefinition(definition)
	if err != nil {
		return ExecutionDefinition{}, fmt.Errorf("validate %q: %w", source, err)
	}
	return normalized, nil
}

// NormalizeExecutionDefinition validates, detaches, and canonicalizes
// unordered declaration sets without changing section, param, or enum order.
func NormalizeExecutionDefinition(definition ExecutionDefinition) (ExecutionDefinition, error) {
	if err := ValidateExecutionDefinition(definition); err != nil {
		return ExecutionDefinition{}, err
	}
	normalized := CloneExecutionDefinition(definition)
	normalized.SupportedTargetTypes = canonicalClosedOrder(normalized.SupportedTargetTypes, []string{TargetTypeDomain, TargetTypeIP, TargetTypeCIDR})
	sort.Strings(normalized.ExecutionResources)
	for sectionIndex := range normalized.ConfigSections {
		for paramIndex := range normalized.ConfigSections[sectionIndex].Params {
			param := &normalized.ConfigSections[sectionIndex].Params[paramIndex]
			if param.Type != ParamTypeInteger || param.Default == nil {
				continue
			}
			value, _ := integerValue(param.Default)
			if value >= int64(minInt()) && value <= int64(maxInt()) {
				param.Default = int(value)
			} else {
				param.Default = value
			}
		}
	}
	return normalized, nil
}

// CloneExecutionDefinition returns an independently mutable copy. Callers
// use the clone to preserve the installed package definition as immutable input.
func CloneExecutionDefinition(definition ExecutionDefinition) ExecutionDefinition {
	cloned := definition
	cloned.SupportedTargetTypes = cloneStrings(definition.SupportedTargetTypes)
	cloned.ExecutionResources = cloneStrings(definition.ExecutionResources)
	if definition.ConfigSections != nil {
		cloned.ConfigSections = make([]ConfigSectionDefinition, len(definition.ConfigSections))
		for sectionIndex, section := range definition.ConfigSections {
			cloned.ConfigSections[sectionIndex] = cloneConfigSectionDefinition(section)
		}
	}
	return cloned
}

func cloneConfigSectionDefinition(section ConfigSectionDefinition) ConfigSectionDefinition {
	cloned := section
	if section.Params != nil {
		cloned.Params = make([]ParamDefinition, len(section.Params))
		for paramIndex, param := range section.Params {
			cloned.Params[paramIndex] = cloneParamDefinition(param)
		}
	}
	return cloned
}

func cloneParamDefinition(param ParamDefinition) ParamDefinition {
	cloned := param
	cloned.Default = deepCopyAny(param.Default)
	cloned.Enum = cloneStrings(param.Enum)
	cloned.Minimum = cloneIntPointer(param.Minimum)
	cloned.Maximum = cloneIntPointer(param.Maximum)
	cloned.MinLength = cloneIntPointer(param.MinLength)
	cloned.MaxLength = cloneIntPointer(param.MaxLength)
	cloned.MinItems = cloneIntPointer(param.MinItems)
	cloned.MaxItems = cloneIntPointer(param.MaxItems)
	if param.Resource != nil {
		resource := *param.Resource
		cloned.Resource = &resource
	}
	return cloned
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func validateClosedStringSet(path string, values []string, allowEmpty bool, allowed []string, label string) error {
	if values == nil {
		return fmt.Errorf("%s is required", path)
	}
	if !allowEmpty && len(values) == 0 {
		return fmt.Errorf("%s must not be empty", path)
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, value := range allowed {
		allowedSet[value] = struct{}{}
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, exists := allowedSet[value]; !exists {
			return fmt.Errorf("unknown %s %q", label, value)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("duplicate %s %q", label, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateParamDefinition(sectionID string, param ParamDefinition) error {
	path := fmt.Sprintf("config section %q param %q", sectionID, param.Key)
	if param.Type != ParamTypeInteger && param.Type != ParamTypeString && param.Type != ParamTypeStringArray && param.Type != ParamTypeBoolean {
		return fmt.Errorf("%s has unsupported param type %q", path, param.Type)
	}
	if param.Resource != nil {
		if param.Type != ParamTypeString {
			return fmt.Errorf("%s has resource binding but type is %s", path, param.Type)
		}
		if param.Resource.Kind != ConfigResourceKindWordlist {
			return fmt.Errorf("%s has unknown resource kind %q", path, param.Resource.Kind)
		}
		defaultValue, ok := param.Default.(string)
		if !ok || strings.TrimSpace(defaultValue) == "" {
			return fmt.Errorf("%s resource binding requires a non-empty default", path)
		}
	}

	switch param.Type {
	case ParamTypeInteger:
		if param.MinLength != nil || param.MaxLength != nil || param.MinItems != nil || param.MaxItems != nil || param.Pattern != "" || param.Enum != nil {
			return fmt.Errorf("%s uses string constraints but type is integer", path)
		}
		if param.Minimum != nil && param.Maximum != nil && *param.Minimum > *param.Maximum {
			return fmt.Errorf("%s has minimum greater than maximum", path)
		}
	case ParamTypeString:
		if param.Minimum != nil || param.Maximum != nil || param.MinItems != nil || param.MaxItems != nil {
			return fmt.Errorf("%s uses minimum/maximum but type is string", path)
		}
		if param.MinLength != nil && *param.MinLength < 0 {
			return fmt.Errorf("%s has negative minLength", path)
		}
		if param.MaxLength != nil && *param.MaxLength < 0 {
			return fmt.Errorf("%s has negative maxLength", path)
		}
		if param.MinLength != nil && param.MaxLength != nil && *param.MinLength > *param.MaxLength {
			return fmt.Errorf("%s has minLength greater than maxLength", path)
		}
		if param.Pattern != "" {
			if _, err := regexp.Compile(param.Pattern); err != nil {
				return fmt.Errorf("%s has invalid pattern: %w", path, err)
			}
		}
		if param.Enum != nil {
			if len(param.Enum) == 0 {
				return fmt.Errorf("%s enum must not be empty", path)
			}
			seen := make(map[string]struct{}, len(param.Enum))
			for _, candidate := range param.Enum {
				if _, exists := seen[candidate]; exists {
					return fmt.Errorf("%s has duplicate enum value %q", path, candidate)
				}
				seen[candidate] = struct{}{}
			}
		}
	case ParamTypeStringArray:
		if param.Minimum != nil || param.Maximum != nil || param.MinLength != nil || param.MaxLength != nil || param.Pattern != "" {
			return fmt.Errorf("%s declares scalar constraints unsupported for stringArray", path)
		}
		if param.MinItems != nil && *param.MinItems < 0 {
			return fmt.Errorf("%s has negative minItems", path)
		}
		if param.MaxItems != nil && *param.MaxItems < 0 {
			return fmt.Errorf("%s has negative maxItems", path)
		}
		if param.MinItems != nil && param.MaxItems != nil && *param.MinItems > *param.MaxItems {
			return fmt.Errorf("%s has minItems greater than maxItems", path)
		}
		if param.Enum != nil {
			if len(param.Enum) == 0 {
				return fmt.Errorf("%s enum must not be empty", path)
			}
			seen := make(map[string]struct{}, len(param.Enum))
			for _, candidate := range param.Enum {
				if _, exists := seen[candidate]; exists {
					return fmt.Errorf("%s has duplicate enum value %q", path, candidate)
				}
				seen[candidate] = struct{}{}
			}
		}
	case ParamTypeBoolean:
		if param.Minimum != nil || param.Maximum != nil || param.MinLength != nil || param.MaxLength != nil || param.MinItems != nil || param.MaxItems != nil || param.Pattern != "" || param.Enum != nil {
			return fmt.Errorf("%s declares constraints unsupported for boolean", path)
		}
	}

	if param.Default == nil {
		return nil
	}
	if err := validateParamValue(path+" default", param.Default, param); err != nil {
		return err
	}
	return nil
}

func validateParamValue(path string, value any, param ParamDefinition) error {
	switch param.Type {
	case ParamTypeInteger:
		integer, ok := integerValue(value)
		if !ok {
			return fmt.Errorf("%s is a non-integer default for integer type", path)
		}
		if param.Minimum != nil && integer < int64(*param.Minimum) {
			return fmt.Errorf("%s below minimum %d", path, *param.Minimum)
		}
		if param.Maximum != nil && integer > int64(*param.Maximum) {
			return fmt.Errorf("%s above maximum %d", path, *param.Maximum)
		}
	case ParamTypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s is a non-boolean default for boolean type", path)
		}
	case ParamTypeString:
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s is a non-string default for string type", path)
		}
		if param.MinLength != nil && utf8.RuneCountInString(text) < *param.MinLength {
			return fmt.Errorf("%s is shorter than minLength %d", path, *param.MinLength)
		}
		if param.MaxLength != nil && utf8.RuneCountInString(text) > *param.MaxLength {
			return fmt.Errorf("%s is longer than maxLength %d", path, *param.MaxLength)
		}
		if param.Pattern != "" {
			pattern, _ := regexp.Compile(param.Pattern)
			if !pattern.MatchString(text) {
				return fmt.Errorf("%s does not match pattern %q", path, param.Pattern)
			}
		}
		if param.Enum != nil && !stringInEnum(text, param.Enum) {
			return fmt.Errorf("%s is not in enum", path)
		}
	case ParamTypeStringArray:
		values, ok := stringArrayValue(value)
		if !ok {
			return fmt.Errorf("%s is a non-string-array default for stringArray type", path)
		}
		if param.MinItems != nil && len(values) < *param.MinItems {
			return fmt.Errorf("%s has fewer than minItems %d", path, *param.MinItems)
		}
		if param.MaxItems != nil && len(values) > *param.MaxItems {
			return fmt.Errorf("%s has more than maxItems %d", path, *param.MaxItems)
		}
		if param.MinItems != nil || param.MaxItems != nil {
			seen := make(map[string]struct{}, len(values))
			for _, text := range values {
				if _, exists := seen[text]; exists {
					return fmt.Errorf("%s contains duplicate value %q", path, text)
				}
				seen[text] = struct{}{}
			}
		}
		for _, text := range values {
			if param.Enum != nil && !stringInEnum(text, param.Enum) {
				return fmt.Errorf("%s contains a value not in enum", path)
			}
		}
	}
	return nil
}

func stringArrayValue(value any) ([]string, bool) {
	switch typed := value.(type) {
	case []string:
		if typed == nil {
			return nil, false
		}
		// Preserve a non-nil empty slice so Profile JSON emits [] rather than null.
		copied := make([]string, len(typed))
		copy(copied, typed)
		return copied, true
	case []any:
		result := make([]string, len(typed))
		for index, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			result[index] = text
		}
		return result, true
	default:
		return nil, false
	}
}

func integerValue(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int8:
		return int64(typed), true
	case int16:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case uint:
		if uint64(typed) > math.MaxInt64 {
			return 0, false
		}
		return int64(typed), true
	case uint8:
		return int64(typed), true
	case uint16:
		return int64(typed), true
	case uint32:
		return int64(typed), true
	case uint64:
		if typed > math.MaxInt64 {
			return 0, false
		}
		return int64(typed), true
	case float32:
		return integralFloat64(float64(typed))
	case float64:
		return integralFloat64(typed)
	case json.Number:
		parsed, err := strconv.ParseInt(typed.String(), 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func integralFloat64(value float64) (int64, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value || value < math.MinInt64 || value >= math.MaxInt64 {
		return 0, false
	}
	return int64(value), true
}

func canonicalClosedOrder(values, canonical []string) []string {
	ordered := make([]string, 0, len(values))
	for _, candidate := range canonical {
		for _, value := range values {
			if value == candidate {
				ordered = append(ordered, value)
				break
			}
		}
	}
	return ordered
}

func consumeExecutionDefinitionJSONEOF(decoder *json.Decoder) error {
	if _, err := decoder.Token(); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return fmt.Errorf("unexpected trailing JSON content")
}

func minInt() int {
	return -maxInt() - 1
}

func maxInt() int {
	return int(^uint(0) >> 1)
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

func stringInEnum(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
