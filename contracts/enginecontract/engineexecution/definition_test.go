package engineexecution

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestConfigIdentityValidatorsMatchEngineGrammar(t *testing.T) {
	for _, value := range []string{"scan", "naabu_active", "a1"} {
		if err := ValidateConfigSectionID(value); err != nil {
			t.Fatalf("ValidateConfigSectionID(%q) error = %v", value, err)
		}
	}
	for _, value := range []string{"threads", "rate-limit", "a1"} {
		if err := ValidateConfigParamKey(value); err != nil {
			t.Fatalf("ValidateConfigParamKey(%q) error = %v", value, err)
		}
	}
	for _, value := range []string{"", "Bad", "bad-section", " bad"} {
		if err := ValidateConfigSectionID(value); err == nil {
			t.Fatalf("ValidateConfigSectionID(%q) succeeded", value)
		}
	}
	for _, value := range []string{"", "Bad", "bad_key", "enabled", " bad"} {
		if err := ValidateConfigParamKey(value); err == nil {
			t.Fatalf("ValidateConfigParamKey(%q) succeeded", value)
		}
	}
}

func validExecutionDefinitionForTest() ExecutionDefinition {
	minimum := 1
	return ExecutionDefinition{
		EngineAPIMajor:       2,
		SupportedTargetTypes: []string{"domain"},
		ConfigSections: []ConfigSectionDefinition{
			{
				ID:             "bruteforce",
				DefaultEnabled: true,
				Params: []ParamDefinition{
					{
						Key:       "wordlist",
						Type:      "string",
						Default:   "subdomains.txt",
						MinLength: &minimum,
						Resource:  &ParamResourceBinding{Kind: "wordlist"},
					},
				},
			},
		},
		ExecutionResources: []string{"subfinderProviderConfig"},
	}
}

func TestExecutionDefinitionValidatesAndClonesWithoutAliasing(t *testing.T) {
	definition := validExecutionDefinitionForTest()
	if err := ValidateExecutionDefinition(definition); err != nil {
		t.Fatalf("ValidateExecutionDefinition failed: %v", err)
	}

	cloned := CloneExecutionDefinition(definition)
	cloned.SupportedTargetTypes[0] = "ip"
	cloned.ConfigSections[0].RequiredEnabled = true
	cloned.ConfigSections[0].Params[0].Default = "other.txt"
	cloned.ConfigSections[0].Params[0].Resource.Kind = "other"
	cloned.ExecutionResources[0] = "other"

	if got := definition.SupportedTargetTypes[0]; got != "domain" {
		t.Fatalf("clone mutated supported target type: %q", got)
	}
	if definition.ConfigSections[0].RequiredEnabled {
		t.Fatalf("clone mutated requiredEnabled declaration")
	}
	if got := definition.ConfigSections[0].Params[0].Default; got != "subdomains.txt" {
		t.Fatalf("clone mutated default: %#v", got)
	}
	if got := definition.ConfigSections[0].Params[0].Resource.Kind; got != "wordlist" {
		t.Fatalf("clone mutated resource: %q", got)
	}
	if got := definition.ExecutionResources[0]; got != "subfinderProviderConfig" {
		t.Fatalf("clone mutated platform resource: %q", got)
	}
}

func TestExecutionDefinitionValidatesStringArrayCardinality(t *testing.T) {
	minimum, maximum := 1, 2
	definition := validExecutionDefinitionForTest()
	definition.ConfigSections[0].Params = append(definition.ConfigSections[0].Params, ParamDefinition{
		Key: "scan-targets", Type: ParamTypeStringArray, Default: []string{"website"},
		Enum: []string{"website", "endpoint"}, MinItems: &minimum, MaxItems: &maximum,
	})
	if err := ValidateExecutionDefinition(definition); err != nil {
		t.Fatal(err)
	}
	cloned := CloneExecutionDefinition(definition)
	*cloned.ConfigSections[0].Params[1].MinItems = 0
	if *definition.ConfigSections[0].Params[1].MinItems != 1 {
		t.Fatal("clone aliased minItems")
	}
	definition.ConfigSections[0].Params[1].Default = []string{}
	if err := ValidateExecutionDefinition(definition); err == nil || !strings.Contains(err.Error(), "fewer than minItems") {
		t.Fatalf("empty cardinality default error = %v", err)
	}
}

func TestResourceDefaultIsANonEmptyLogicalPreferredCandidate(t *testing.T) {
	definition := validExecutionDefinitionForTest()
	definition.ConfigSections[0].Params[0].Default = "deployment-may-not-have-this.txt"
	if err := ValidateExecutionDefinition(definition); err != nil {
		t.Fatalf("logical preferred candidate must not require deployment lookup: %v", err)
	}
	definition.ConfigSections[0].Params[0].Default = " "
	if err := ValidateExecutionDefinition(definition); err == nil || !strings.Contains(err.Error(), "non-empty default") {
		t.Fatalf("empty resource default error = %v", err)
	}
}

func TestValidateExecutionDefinitionAcceptsCanonicalSnakeCaseSection(t *testing.T) {
	definition := validExecutionDefinitionForTest()
	definition.ConfigSections[0].ID = "naabu_active"
	if err := ValidateExecutionDefinition(definition); err != nil {
		t.Fatalf("ValidateExecutionDefinition rejected canonical section ID: %v", err)
	}
}

func TestDecodeExecutionDefinitionStrictlyNormalizesCompleteDefinition(t *testing.T) {
	payload := []byte(`{
  "engineApiMajor": 2,
  "supportedTargetTypes": ["cidr", "domain", "ip"],
  "configSections": [{
    "id": "httpx",
    "defaultEnabled": true,
    "requiredEnabled": true,
    "params": [{
      "key": "request-timeout",
      "type": "integer",
      "default": 10,
      "minimum": 1,
      "maximum": 60
    }, {
      "key": "mode",
      "type": "string",
      "default": "fast",
      "minLength": 2,
      "maxLength": 8,
      "pattern": "^[a-z]+$",
      "enum": ["fast", "safe"]
    }, {
      "key": "wordlist",
      "type": "string",
      "default": "names.txt",
      "resource": {"kind": "wordlist"}
    }]
  }],
  "executionResources": []
}`)

	definition, err := DecodeExecutionDefinition(payload, "fixture")
	if err != nil {
		t.Fatalf("DecodeExecutionDefinition failed: %v", err)
	}
	if got, want := definition.SupportedTargetTypes, []string{"domain", "ip", "cidr"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("supported target types = %#v, want %#v", got, want)
	}
	if definition.ExecutionResources == nil || len(definition.ExecutionResources) != 0 {
		t.Fatalf("explicit empty executionResources was not preserved: %#v", definition.ExecutionResources)
	}
	if !definition.ConfigSections[0].RequiredEnabled {
		t.Fatalf("requiredEnabled declaration was lost: %#v", definition.ConfigSections[0])
	}
	requestTimeout := definition.ConfigSections[0].Params[0]
	if requestTimeout.Default != 10 || requestTimeout.Minimum == nil || *requestTimeout.Minimum != 1 || requestTimeout.Maximum == nil || *requestTimeout.Maximum != 60 {
		t.Fatalf("integer declaration was lost: %#v", requestTimeout)
	}
	mode := definition.ConfigSections[0].Params[1]
	if mode.MinLength == nil || *mode.MinLength != 2 || mode.MaxLength == nil || *mode.MaxLength != 8 || mode.Pattern != "^[a-z]+$" || !reflect.DeepEqual(mode.Enum, []string{"fast", "safe"}) {
		t.Fatalf("string declaration was lost: %#v", mode)
	}
	resource := definition.ConfigSections[0].Params[2].Resource
	if resource == nil || resource.Kind != ConfigResourceKindWordlist {
		t.Fatalf("resource declaration was lost: %#v", resource)
	}

	roundtripPayload, err := json.Marshal(definition)
	if err != nil {
		t.Fatalf("marshal normalized definition: %v", err)
	}
	roundtrip, err := DecodeExecutionDefinition(roundtripPayload, "roundtrip")
	if err != nil {
		t.Fatalf("roundtrip DecodeExecutionDefinition failed: %v", err)
	}
	if !reflect.DeepEqual(roundtrip, definition) {
		t.Fatalf("roundtrip lost definition fields:\n got: %#v\nwant: %#v", roundtrip, definition)
	}
}

func TestDecodeExecutionDefinitionPreservesOptionalExecutionResourceOmission(t *testing.T) {
	payload := []byte(`{
  "engineApiMajor": 2,
  "supportedTargetTypes": ["domain"],
  "configSections": [{"id":"scan","params":[{"key":"enabled-mode","type":"boolean","default":false}]}]
}`)

	definition, err := DecodeExecutionDefinition(payload, "fixture")
	if err != nil {
		t.Fatalf("DecodeExecutionDefinition failed: %v", err)
	}
	if definition.ExecutionResources != nil {
		t.Fatalf("omitted executionResources became present: %#v", definition.ExecutionResources)
	}
	cloned := CloneExecutionDefinition(definition)
	if cloned.ExecutionResources != nil {
		t.Fatalf("clone changed omitted executionResources: %#v", cloned.ExecutionResources)
	}
}

func TestDecodeExecutionDefinitionRejectsExplicitNullDeclarations(t *testing.T) {
	base := `{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","defaultEnabled":true,"params":[{"key":"threads","type":"integer","default":2,"minimum":1,"maximum":3},{"key":"wordlist","type":"string","default":"fast","minLength":1,"maxLength":4,"pattern":"^[a-z]+$","enum":["fast"],"resource":{"kind":"wordlist"}}]}]}`
	cases := []struct {
		name          string
		old           string
		replacement   string
		wantSubstring string
	}{
		{name: "integer default", old: `"default":2`, replacement: `"default":null`, wantSubstring: "null JSON value"},
		{name: "string default", old: `"default":"fast"`, replacement: `"default":null`, wantSubstring: "null JSON value"},
		{name: "default enabled", old: `"defaultEnabled":true`, replacement: `"defaultEnabled":null`, wantSubstring: "null JSON value"},
		{name: "required enabled", old: `"defaultEnabled":true`, replacement: `"defaultEnabled":true,"requiredEnabled":null`, wantSubstring: "null JSON value"},
		{name: "minimum", old: `"minimum":1`, replacement: `"minimum":null`, wantSubstring: "null JSON value"},
		{name: "maximum", old: `"maximum":3`, replacement: `"maximum":null`, wantSubstring: "null JSON value"},
		{name: "min length", old: `"minLength":1`, replacement: `"minLength":null`, wantSubstring: "null JSON value"},
		{name: "max length", old: `"maxLength":4`, replacement: `"maxLength":null`, wantSubstring: "null JSON value"},
		{name: "pattern", old: `"pattern":"^[a-z]+$"`, replacement: `"pattern":null`, wantSubstring: "null JSON value"},
		{name: "enum", old: `"enum":["fast"]`, replacement: `"enum":null`, wantSubstring: "null JSON value"},
		{name: "resource", old: `"resource":{"kind":"wordlist"}`, replacement: `"resource":null`, wantSubstring: "null JSON value"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := strings.Replace(base, tc.old, tc.replacement, 1)
			_, err := DecodeExecutionDefinition([]byte(payload), "fixture")
			if err == nil || !strings.Contains(err.Error(), tc.wantSubstring) {
				t.Fatalf("expected explicit null rejection, got %v", err)
			}
		})
	}
}

func TestDecodeExecutionDefinitionRejectsUnknownAndTrailingJSON(t *testing.T) {
	valid := `{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","params":[{"key":"mode","type":"string"}]}]}`
	cases := []struct {
		name    string
		payload string
		message string
	}{
		{"unknown execution field", `{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","params":[{"key":"mode","type":"string"}]}],"runtimeRef":"runtime.demo"}`, "unknown field"},
		{"unknown section field", `{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","required":true,"params":[{"key":"mode","type":"string"}]}]}`, "unknown field"},
		{"conditional required section", `{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","defaultEnabled":true,"requiredEnabled":true,"when":"recon.enabled","params":[{"key":"mode","type":"string"}]}]}`, "unknown field"},
		{"group required section", `{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","defaultEnabled":true,"requiredEnabled":true,"group":"discovery","params":[{"key":"mode","type":"string"}]}]}`, "unknown field"},
		{"orchestration required section", `{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","defaultEnabled":true,"requiredEnabled":true,"after":"recon","params":[{"key":"mode","type":"string"}]}]}`, "unknown field"},
		{"unknown param field", `{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","params":[{"key":"mode","type":"string","source":"task"}]}]}`, "unknown field"},
		{"unknown resource field", `{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","params":[{"key":"wordlist","type":"string","default":"names.txt","resource":{"kind":"wordlist","path":"/tmp/x"}}]}]}`, "unknown field"},
		{"null execution resources", `{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","params":[{"key":"mode","type":"string"}]}],"executionResources":null}`, "executionResources"},
		{"duplicate execution field", `{"engineApiMajor":2,"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","params":[{"key":"mode","type":"string"}]}]}`, "duplicate JSON field"},
		{"duplicate resource field", `{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","params":[{"key":"wordlist","type":"string","default":"names.txt","resource":{"kind":"wordlist","kind":"wordlist"}}]}]}`, "duplicate JSON field"},
		{"trailing JSON", valid + `{}`, "unexpected trailing JSON content"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeExecutionDefinition([]byte(tc.payload), "fixture")
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("expected %q error, got %v", tc.message, err)
			}
		})
	}
}

func TestValidateExecutionDefinitionRejectsInvalidDeclaration(t *testing.T) {
	minimum := 1
	maximum := 10
	minLength := 1
	maxLength := 32
	valid := validExecutionDefinitionForTest()
	valid.ConfigSections[0].Params[0].Minimum = nil
	valid.ConfigSections[0].Params[0].MinLength = &minLength
	valid.ConfigSections[0].Params[0].MaxLength = &maxLength
	valid.ConfigSections[0].Params[0].Pattern = `^[a-z.]+$`
	valid.ConfigSections[0].Params = append(valid.ConfigSections[0].Params, ParamDefinition{
		Key: "threads", Type: "integer", Default: 5, Minimum: &minimum, Maximum: &maximum,
	})

	cases := []struct {
		name    string
		mutate  func(*ExecutionDefinition)
		message string
	}{
		{"engine API major", func(def *ExecutionDefinition) { def.EngineAPIMajor = 0 }, "engineApiMajor"},
		{"missing target types", func(def *ExecutionDefinition) { def.SupportedTargetTypes = nil }, "supportedTargetTypes"},
		{"empty target types", func(def *ExecutionDefinition) { def.SupportedTargetTypes = []string{} }, "must not be empty"},
		{"unknown target type", func(def *ExecutionDefinition) { def.SupportedTargetTypes = []string{"hostname"} }, "unknown supported target type"},
		{"target type alias", func(def *ExecutionDefinition) { def.SupportedTargetTypes = []string{" DOMAIN"} }, "unknown supported target type"},
		{"duplicate target type", func(def *ExecutionDefinition) { def.SupportedTargetTypes = []string{"domain", "domain"} }, "duplicate supported target type"},
		{"unknown execution resource", func(def *ExecutionDefinition) { def.ExecutionResources = []string{"providerConfig"} }, "unknown execution resource"},
		{"duplicate execution resource", func(def *ExecutionDefinition) {
			def.ExecutionResources = []string{"subfinderProviderConfig", "subfinderProviderConfig"}
		}, "duplicate execution resource"},
		{"execution whitespace alias", func(def *ExecutionDefinition) { def.ExecutionResources = []string{" subfinderProviderConfig"} }, "unknown execution resource"},
		{"section id format", func(def *ExecutionDefinition) { def.ConfigSections[0].ID = "BruteForce" }, "config section id"},
		{"section kebab alias", func(def *ExecutionDefinition) { def.ConfigSections[0].ID = "naabu-active" }, "config section id"},
		{"empty config sections", func(def *ExecutionDefinition) { def.ConfigSections = nil }, "configSections"},
		{"duplicate section id", func(def *ExecutionDefinition) {
			def.ConfigSections = append(def.ConfigSections, def.ConfigSections[0])
		}, "duplicate config section id"},
		{"required section lacks default", func(def *ExecutionDefinition) {
			def.ConfigSections[0].RequiredEnabled = true
			def.ConfigSections[0].DefaultEnabled = false
		}, "requiredEnabled requires defaultEnabled"},
		{"empty section params", func(def *ExecutionDefinition) { def.ConfigSections[0].Params = nil }, "must define at least one param"},
		{"param key format", func(def *ExecutionDefinition) { def.ConfigSections[0].Params[0].Key = "word_list" }, "param key"},
		{"reserved param key", func(def *ExecutionDefinition) { def.ConfigSections[0].Params[0].Key = "enabled" }, "must not be enabled"},
		{"duplicate param key", func(def *ExecutionDefinition) {
			def.ConfigSections[0].Params = append(def.ConfigSections[0].Params, def.ConfigSections[0].Params[0])
		}, "duplicate param key"},
		{"missing param type", func(def *ExecutionDefinition) { def.ConfigSections[0].Params[0].Type = "" }, "unsupported param type"},
		{"param type alias", func(def *ExecutionDefinition) { def.ConfigSections[0].Params[0].Type = "String" }, "unsupported param type"},
		{"integer bounds on string", func(def *ExecutionDefinition) {
			def.ConfigSections[0].Params[0].Type = "string"
			def.ConfigSections[0].Params[0].Minimum = &minimum
		}, "minimum/maximum"},
		{"integer bound order", func(def *ExecutionDefinition) {
			def.ConfigSections[0].Params[1].Minimum = &maximum
			def.ConfigSections[0].Params[1].Maximum = &minimum
		}, "minimum greater"},
		{"boolean constraints", func(def *ExecutionDefinition) {
			def.ConfigSections[0].Params[1].Type = "boolean"
			def.ConfigSections[0].Params[1].Minimum = nil
			def.ConfigSections[0].Params[1].Maximum = nil
			def.ConfigSections[0].Params[1].Enum = []string{"true"}
			def.ConfigSections[0].Params[1].Default = true
		}, "unsupported for boolean"},
		{"negative length", func(def *ExecutionDefinition) { value := -1; def.ConfigSections[0].Params[0].MinLength = &value }, "negative minLength"},
		{"length order", func(def *ExecutionDefinition) {
			def.ConfigSections[0].Params[0].MinLength = &maxLength
			def.ConfigSections[0].Params[0].MaxLength = &minLength
		}, "minLength greater"},
		{"invalid pattern", func(def *ExecutionDefinition) { def.ConfigSections[0].Params[0].Pattern = "[" }, "invalid pattern"},
		{"empty enum", func(def *ExecutionDefinition) { def.ConfigSections[0].Params[0].Enum = []string{} }, "enum must not be empty"},
		{"duplicate enum", func(def *ExecutionDefinition) {
			def.ConfigSections[0].Params[0].Enum = []string{"subdomains.txt", "subdomains.txt"}
		}, "duplicate enum value"},
		{"resource type", func(def *ExecutionDefinition) { def.ConfigSections[0].Params[0].Type = "integer" }, "resource binding"},
		{"resource kind", func(def *ExecutionDefinition) { def.ConfigSections[0].Params[0].Resource.Kind = "resolver" }, "unknown resource kind"},
		{"resource default empty", func(def *ExecutionDefinition) { def.ConfigSections[0].Params[0].Default = " " }, "non-empty default"},
		{"wrong integer default", func(def *ExecutionDefinition) { def.ConfigSections[0].Params[1].Default = "1" }, "non-integer default"},
		{"default below minimum", func(def *ExecutionDefinition) { def.ConfigSections[0].Params[1].Default = 0 }, "default below minimum"},
		{"default above maximum", func(def *ExecutionDefinition) { def.ConfigSections[0].Params[1].Default = 11 }, "default above maximum"},
		{"string default outside enum", func(def *ExecutionDefinition) {
			def.ConfigSections[0].Params[0].Enum = []string{"other.txt"}
		}, "default is not in enum"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			definition := CloneExecutionDefinition(valid)
			tc.mutate(&definition)
			err := ValidateExecutionDefinition(definition)
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("expected %q error, got %v", tc.message, err)
			}
		})
	}
}
