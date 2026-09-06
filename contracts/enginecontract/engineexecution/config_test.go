package engineexecution

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

func configExecutionDefinitionForTest() ExecutionDefinition {
	minimum := 1
	maximum := 20
	minLength := 2
	maxLength := 8
	resourceMinLength := 1
	return ExecutionDefinition{
		EngineAPIMajor:       2,
		SupportedTargetTypes: []string{TargetTypeDomain},
		ConfigSections: []ConfigSectionDefinition{
			{
				ID:             "scan",
				DefaultEnabled: true,
				Params: []ParamDefinition{
					{Key: "threads", Type: ParamTypeInteger, Default: 10, Minimum: &minimum, Maximum: &maximum},
					{Key: "mode", Type: ParamTypeString, Default: "fast", Enum: []string{"fast", "safe"}},
					{Key: "filters", Type: ParamTypeStringArray, Default: []string{}, Enum: []string{"fast", "safe"}},
					{Key: "name", Type: ParamTypeString, MinLength: &minLength, MaxLength: &maxLength, Pattern: `^[a-z]+$`},
					{Key: "passive", Type: ParamTypeBoolean, Default: false},
				},
			},
			{
				ID: "optional",
				Params: []ParamDefinition{
					{
						Key:       "wordlist",
						Type:      ParamTypeString,
						Default:   "names.txt",
						MinLength: &resourceMinLength,
						Pattern:   `^[a-z.]+$`,
						Resource:  &ParamResourceBinding{Kind: ConfigResourceKindWordlist},
					},
					{Key: "timeout", Type: ParamTypeInteger, Default: 5, Minimum: &minimum},
					{Key: "label", Type: ParamTypeString},
				},
			},
		},
	}
}

func validConfigForTest() map[string]any {
	return map[string]any{
		"scan": map[string]any{
			"enabled": true,
			"name":    "demo",
		},
		"optional": map[string]any{
			"enabled": false,
		},
	}
}

func TestNormalizeAndValidateConfigAppliesDefaultsWithoutMutatingInput(t *testing.T) {
	raw := validConfigForTest()
	normalized, err := NormalizeAndValidateConfig(raw, configExecutionDefinitionForTest())
	if err != nil {
		t.Fatalf("NormalizeAndValidateConfig failed: %v", err)
	}

	scan := normalized["scan"].(map[string]any)
	if scan["threads"] != 10 || scan["mode"] != "fast" || scan["passive"] != false || scan["name"] != "demo" {
		t.Fatalf("scan defaults or explicit values were lost: %#v", scan)
	}
	optional := normalized["optional"].(map[string]any)
	if optional["enabled"] != false || len(optional) != 1 {
		t.Fatalf("disabled section must retain only its switch: %#v", optional)
	}
	if _, exists := optional["label"]; exists {
		t.Fatalf("disabled section received a missing non-resource value without a default: %#v", optional)
	}
	if _, exists := raw["scan"].(map[string]any)["threads"]; exists {
		t.Fatalf("normalization mutated the caller input: %#v", raw)
	}

	normalized["scan"].(map[string]any)["name"] = "changed"
	if got := raw["scan"].(map[string]any)["name"]; got != "demo" {
		t.Fatalf("normalized config aliases caller input: %#v", got)
	}
}

func TestNormalizeAndValidateConfigPreservesStringArrayOrderAndDuplicates(t *testing.T) {
	raw := validConfigForTest()
	raw["scan"].(map[string]any)["filters"] = []any{"safe", "fast", "safe"}

	normalized, err := NormalizeAndValidateConfig(raw, configExecutionDefinitionForTest())
	if err != nil {
		t.Fatalf("NormalizeAndValidateConfig failed: %v", err)
	}
	values, ok := normalized["scan"].(map[string]any)["filters"].([]string)
	if !ok || len(values) != 3 || values[0] != "safe" || values[1] != "fast" || values[2] != "safe" {
		t.Fatalf("string array was not preserved: %#v", normalized)
	}
	if _, err := NormalizeAndValidateConfig(map[string]any{"scan": map[string]any{"enabled": true, "name": "demo", "filters": []any{"unknown"}}}, configExecutionDefinitionForTest()); err == nil || !strings.Contains(err.Error(), "scan.filters contains a value not in") {
		t.Fatalf("string array enum error = %v", err)
	}
}

func TestNormalizeAndValidateConfigEnforcesCardinalityAndCanonicalEnumOrder(t *testing.T) {
	definition := configExecutionDefinitionForTest()
	minimum, maximum := 1, 2
	definition.ConfigSections[0].Params[2].MinItems = &minimum
	definition.ConfigSections[0].Params[2].MaxItems = &maximum
	definition.ConfigSections[0].Params[2].Default = []string{"fast"}

	raw := validConfigForTest()
	raw["scan"].(map[string]any)["filters"] = []any{"safe", "fast"}
	normalized, err := NormalizeAndValidateConfig(raw, definition)
	if err != nil {
		t.Fatal(err)
	}
	if got := normalized["scan"].(map[string]any)["filters"]; !reflect.DeepEqual(got, []string{"fast", "safe"}) {
		t.Fatalf("cardinality enum order = %#v, want manifest order", got)
	}
	for name, values := range map[string][]any{
		"empty":     {},
		"duplicate": {"fast", "fast"},
	} {
		t.Run(name, func(t *testing.T) {
			config := validConfigForTest()
			config["scan"].(map[string]any)["filters"] = values
			if _, err := NormalizeAndValidateConfig(config, definition); err == nil {
				t.Fatal("invalid cardinality configuration was accepted")
			}
		})
	}
}

func TestNormalizeAndValidateConfigUsesExplicitResourceInsteadOfDefault(t *testing.T) {
	raw := validConfigForTest()
	raw["optional"].(map[string]any)["enabled"] = true
	raw["optional"].(map[string]any)["wordlist"] = "custom.txt"
	raw["optional"].(map[string]any)["timeout"] = json.Number("3")
	raw["optional"].(map[string]any)["label"] = "demo"

	normalized, err := NormalizeAndValidateConfig(raw, configExecutionDefinitionForTest())
	if err != nil {
		t.Fatalf("NormalizeAndValidateConfig failed: %v", err)
	}
	if got := normalized["optional"].(map[string]any)["wordlist"]; got != "custom.txt" {
		t.Fatalf("explicit resource was replaced by default: %#v", got)
	}
}

func TestNormalizeAndValidateConfigRejectsMissingConfig(t *testing.T) {
	_, err := NormalizeAndValidateConfig(nil, configExecutionDefinitionForTest())
	if err == nil || err.Error() != "config is required" {
		t.Fatalf("expected config required error, got %v", err)
	}
}

func TestNormalizeAndValidateConfigBuildsCompleteConfigFromEngineDefaults(t *testing.T) {
	definition := configExecutionDefinitionForTest()
	definition.ConfigSections[0].Params[3].Default = "demo"
	definition.ConfigSections[1].Params[2].Default = "label"
	definition.ConfigSections[0].Params[3].Default = "demo"
	normalized, err := NormalizeAndValidateConfig(map[string]any{}, definition)
	if err != nil {
		t.Fatalf("NormalizeAndValidateConfig failed: %v", err)
	}

	scan := normalized["scan"].(map[string]any)
	if scan["enabled"] != true || scan["threads"] != 10 || scan["mode"] != "fast" || scan["name"] != "demo" || scan["passive"] != false {
		t.Fatalf("default-enabled section was not completed from definition: %#v", scan)
	}
	optional := normalized["optional"].(map[string]any)
	if optional["enabled"] != false || len(optional) != 1 {
		t.Fatalf("disabled section defaults/parameters were not discarded: %#v", optional)
	}
}

func TestNormalizeAndValidateConfigRejectsInvalidRuntimeConfig(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(map[string]any)
		message string
	}{
		{"unknown section", func(config map[string]any) { config["other"] = map[string]any{"enabled": false} }, "unknown config section"},
		{"section object", func(config map[string]any) { config["scan"] = true }, "scan must be object"},
		{"enabled type", func(config map[string]any) { config["scan"].(map[string]any)["enabled"] = "true" }, "scan.enabled must be boolean"},
		{"unknown param", func(config map[string]any) { config["scan"].(map[string]any)["other"] = true }, "scan.other is not allowed"},
		{"enabled missing no-default param", func(config map[string]any) { delete(config["scan"].(map[string]any), "name") }, "scan.name is required when enabled"},
		{"integer type", func(config map[string]any) { config["scan"].(map[string]any)["threads"] = "10" }, "scan.threads must be integer"},
		{"fractional integer", func(config map[string]any) { config["scan"].(map[string]any)["threads"] = 1.5 }, "scan.threads must be integer"},
		{"out of range integer", func(config map[string]any) { config["scan"].(map[string]any)["threads"] = math.Exp2(63) }, "scan.threads must be integer"},
		{"boolean type", func(config map[string]any) { config["scan"].(map[string]any)["passive"] = 1 }, "scan.passive must be boolean"},
		{"string type", func(config map[string]any) { config["scan"].(map[string]any)["name"] = true }, "scan.name must be string"},
		{"enum", func(config map[string]any) { config["scan"].(map[string]any)["mode"] = "turbo" }, "scan.mode must be one of"},
		{"minimum", func(config map[string]any) { config["scan"].(map[string]any)["threads"] = 0 }, "scan.threads must be >= 1"},
		{"maximum", func(config map[string]any) { config["scan"].(map[string]any)["threads"] = 21 }, "scan.threads must be <= 20"},
		{"min length", func(config map[string]any) { config["scan"].(map[string]any)["name"] = "a" }, "scan.name length must be >= 2"},
		{"max length", func(config map[string]any) { config["scan"].(map[string]any)["name"] = "toolongxx" }, "scan.name length must be <= 8"},
		{"pattern", func(config map[string]any) { config["scan"].(map[string]any)["name"] = "BAD" }, "scan.name must match pattern"},
		{"explicit empty resource", func(config map[string]any) {
			config["optional"].(map[string]any)["enabled"] = true
			config["optional"].(map[string]any)["wordlist"] = ""
		}, "optional.wordlist must be non-empty"},
		{"explicit blank resource", func(config map[string]any) {
			config["optional"].(map[string]any)["enabled"] = true
			config["optional"].(map[string]any)["wordlist"] = "  "
		}, "optional.wordlist must be non-empty"},
		{"explicit resource type", func(config map[string]any) {
			config["optional"].(map[string]any)["enabled"] = true
			config["optional"].(map[string]any)["wordlist"] = 10
		}, "optional.wordlist must be string"},
		{"disabled section with parameter", func(config map[string]any) { config["optional"].(map[string]any)["timeout"] = 0 }, "optional disabled config section must contain only enabled"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := validConfigForTest()
			tc.mutate(config)
			_, err := NormalizeAndValidateConfig(config, configExecutionDefinitionForTest())
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("expected %q error, got %v", tc.message, err)
			}
		})
	}
}

func TestNormalizeAndValidateConfigRejectsInvalidDefinition(t *testing.T) {
	definition := configExecutionDefinitionForTest()
	definition.ConfigSections[0].Params[0].Default = "10"

	_, err := NormalizeAndValidateConfig(validConfigForTest(), definition)
	if err == nil || !strings.Contains(err.Error(), "invalid execution definition") || !strings.Contains(err.Error(), "non-integer default") {
		t.Fatalf("expected invalid definition error, got %v", err)
	}
}

func TestValidateCompleteConfigDoesNotFillDefaults(t *testing.T) {
	definition := configExecutionDefinitionForTest()
	definition.ConfigSections[0].Params[3].Default = "demo"
	definition.ConfigSections[1].Params[2].Default = "label"
	defaults, err := NormalizeAndValidateConfig(map[string]any{}, definition)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateCompleteConfig(map[string]any{}, definition); err == nil {
		t.Fatal("complete validation must reject missing sections")
	}
	if _, err := ValidateCompleteConfig(map[string]any{"scan": map[string]any{"enabled": true}}, definition); err == nil {
		t.Fatal("complete validation must reject missing parameters")
	}
	validated, err := ValidateCompleteConfig(defaults, definition)
	if err != nil {
		t.Fatalf("ValidateCompleteConfig(defaults) error = %v", err)
	}
	if validated["scan"].(map[string]any)["threads"] != 10 {
		t.Fatalf("unexpected complete config: %+v", validated)
	}
}

func TestNormalizeAndValidateConfigDoesNotFallbackExplicitInvalidResource(t *testing.T) {
	raw := validConfigForTest()
	optional := raw["optional"].(map[string]any)
	optional["wordlist"] = ""
	optional["enabled"] = true

	_, err := NormalizeAndValidateConfig(raw, configExecutionDefinitionForTest())
	if err == nil || !strings.Contains(err.Error(), "optional.wordlist must be non-empty") {
		t.Fatalf("expected explicit resource rejection without default fallback, got %v", err)
	}
}

func TestNormalizeAndValidateConfigRejectsAllDisabledSections(t *testing.T) {
	raw := validConfigForTest()
	raw["scan"] = map[string]any{"enabled": false}

	_, err := NormalizeAndValidateConfig(raw, configExecutionDefinitionForTest())
	if err == nil || !strings.Contains(err.Error(), "at least one config section must be enabled") {
		t.Fatalf("expected all-disabled section rejection, got %v", err)
	}
}

func TestValidateCompleteConfigRejectsParameterBearingDisabledSection(t *testing.T) {
	raw := map[string]any{
		"scan": map[string]any{
			"enabled": true, "threads": 10, "mode": "fast", "filters": []string{}, "name": "demo", "passive": false,
		},
		"optional": map[string]any{"enabled": false, "timeout": 3},
	}

	_, err := ValidateCompleteConfig(raw, configExecutionDefinitionForTest())
	if err == nil || !strings.Contains(err.Error(), "optional disabled config section must contain only enabled") {
		t.Fatalf("expected disabled section parameter rejection, got %v", err)
	}
}

func TestValidateCompleteConfigRejectsDisabledRequiredSectionWithoutDefaultRepair(t *testing.T) {
	definition := configExecutionDefinitionForTest()
	definition.ConfigSections[0].RequiredEnabled = true
	defaults, err := NormalizeAndValidateConfig(validConfigForTest(), definition)
	if err != nil {
		t.Fatalf("NormalizeAndValidateConfig() error = %v", err)
	}
	defaults["scan"].(map[string]any)["enabled"] = false

	_, err = ValidateCompleteConfig(defaults, definition)
	if err == nil || !strings.Contains(err.Error(), "requiredEnabled") {
		t.Fatalf("ValidateCompleteConfig() error = %v, want requiredEnabled rejection", err)
	}
	if defaults["scan"].(map[string]any)["enabled"] != false {
		t.Fatalf("ValidateCompleteConfig repaired disabled required section: %#v", defaults["scan"])
	}
}
