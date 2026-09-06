package configuration

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestExtractStepConfigurationsAcceptsEnabledAndDisabledBranches(t *testing.T) {
	configs, err := ExtractStepConfigurations(map[string]any{
		"steps": map[string]any{
			"subdomain_discovery": map[string]any{
				"enabled": true,
				"engineConfig": map[string]any{
					"recon": map[string]any{"enabled": true, "timeout": 3600},
				},
			},
			"port_scan": map[string]any{"enabled": false},
		},
	}, map[string]struct{}{"subdomain_discovery": {}, "port_scan": {}})
	if err != nil {
		t.Fatalf("ExtractStepConfigurations failed: %v", err)
	}
	if !configs["subdomain_discovery"].Enabled || configs["port_scan"].Enabled {
		t.Fatalf("unexpected enabled states: %#v", configs)
	}
	recon, ok := configs["subdomain_discovery"].EngineConfig["recon"].(map[string]any)
	if !ok || recon["enabled"] != true || recon["timeout"] != 3600 {
		t.Fatalf("unexpected configs: %#v", configs)
	}
	if configs["port_scan"].EngineConfig != nil {
		t.Fatalf("disabled Step must not carry engineConfig: %#v", configs["port_scan"])
	}
}

func TestEncodeStepConfigurationsDropsDisabledEngineDrafts(t *testing.T) {
	encoded := EncodeStepConfigurations(map[string]StepConfiguration{
		"enabled-step": {
			Enabled:      true,
			EngineConfig: map[string]any{"scan": map[string]any{"enabled": true}},
		},
		"disabled-step": {
			Enabled:      false,
			EngineConfig: map[string]any{"should": "not persist"},
		},
	})

	steps, ok := encoded["steps"].(map[string]any)
	if !ok {
		t.Fatalf("encoded steps has unexpected type: %#v", encoded["steps"])
	}
	if got := steps["disabled-step"]; got == nil {
		t.Fatal("encoded disabled Step is missing")
	} else if entry, ok := got.(map[string]any); !ok || len(entry) != 1 || entry["enabled"] != false {
		t.Fatalf("disabled Step must encode as enabled:false only: %#v", got)
	}
	if entry, ok := steps["enabled-step"].(map[string]any); !ok || entry["enabled"] != true || entry["engineConfig"] == nil {
		t.Fatalf("enabled Step lost Engine configuration: %#v", steps["enabled-step"])
	}
}

func TestExtractStepConfigurationsRejectsEmptyConfiguration(t *testing.T) {
	for _, configuration := range []any{nil, map[string]any{}} {
		_, err := ExtractStepConfigurations(configuration, map[string]struct{}{"subdomain_discovery": {}})
		if err == nil {
			t.Fatalf("ExtractStepConfigurations(%#v) succeeded", configuration)
		}
	}
}

func TestExtractStepConfigurationsRejectsInvalidShape(t *testing.T) {
	tests := map[string]struct {
		configuration any
		message       string
	}{
		"invalid root": {
			configuration: []any{"invalid"},
			message:       "configuration must be an object",
		},
		"unknown top-level field": {
			configuration: map[string]any{"workflow": "default"},
			message:       "configuration must contain only top-level field steps",
		},
		"steps is not object": {
			configuration: map[string]any{"steps": []any{}},
			message:       "configuration.steps must be object",
		},
		"unknown step": {
			configuration: map[string]any{"steps": map[string]any{"unknown": map[string]any{"enabled": false}}},
			message:       "unknown workflow step",
		},
		"step entry is not object": {
			configuration: map[string]any{"steps": map[string]any{"subdomain_discovery": []any{}}},
			message:       "configuration.steps.subdomain_discovery must be object",
		},
		"missing enabled": {
			configuration: map[string]any{"steps": map[string]any{"subdomain_discovery": map[string]any{"engineConfig": map[string]any{}}}},
			message:       "enabled is required",
		},
		"malformed enabled": {
			configuration: map[string]any{"steps": map[string]any{"subdomain_discovery": map[string]any{"enabled": "true", "engineConfig": map[string]any{}}}},
			message:       "enabled must be boolean",
		},
		"enabled without engine config": {
			configuration: map[string]any{"steps": map[string]any{"subdomain_discovery": map[string]any{"enabled": true}}},
			message:       "engineConfig is required when enabled",
		},
		"disabled with engine config": {
			configuration: map[string]any{"steps": map[string]any{"subdomain_discovery": map[string]any{"enabled": false, "engineConfig": map[string]any{}}}},
			message:       "disabled branch only supports enabled",
		},
		"enabled unknown sibling": {
			configuration: map[string]any{"steps": map[string]any{"subdomain_discovery": map[string]any{"enabled": true, "engineConfig": map[string]any{}, "legacy": true}}},
			message:       "enabled branch requires only enabled and engineConfig",
		},
		"engine config is not object": {
			configuration: map[string]any{"steps": map[string]any{"subdomain_discovery": map[string]any{"enabled": true, "engineConfig": []any{}}}},
			message:       "configuration.steps.subdomain_discovery.engineConfig must be object",
		},
		"missing known step": {
			configuration: map[string]any{"steps": map[string]any{"subdomain_discovery": map[string]any{"enabled": false}}},
			message:       "configuration.steps must contain exactly every workflow step",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			knownSteps := map[string]struct{}{"subdomain_discovery": {}}
			if name == "missing known step" {
				knownSteps["port_scan"] = struct{}{}
			}
			_, err := ExtractStepConfigurations(tc.configuration, knownSteps)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("expected %q, got %v", tc.message, err)
			}
		})
	}
}

func TestExtractStepConfigurationsNormalizesYAMLMaps(t *testing.T) {
	configs, err := ExtractStepConfigurations(map[any]any{
		"steps": map[any]any{
			"subdomain_discovery": map[any]any{
				"enabled": true,
				"engineConfig": map[any]any{
					"recon": map[any]any{"enabled": true},
				},
			},
		},
	}, map[string]struct{}{"subdomain_discovery": {}})
	if err != nil {
		t.Fatalf("ExtractStepConfigurations failed: %v", err)
	}
	recon, ok := configs["subdomain_discovery"].EngineConfig["recon"].(map[string]any)
	if !ok || recon["enabled"] != true {
		t.Fatalf("unexpected normalized configs: %#v", configs)
	}
}

func TestExtractStepConfigurationsRoundTripsJSONAndYAMLObjects(t *testing.T) {
	original := map[string]any{
		"steps": map[string]any{
			"first":  map[string]any{"enabled": true, "engineConfig": map[string]any{"scan": map[string]any{"enabled": true}}},
			"second": map[string]any{"enabled": false},
		},
	}
	jsonPayload, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var jsonObject map[string]any
	if err := json.Unmarshal(jsonPayload, &jsonObject); err != nil {
		t.Fatal(err)
	}
	if _, err := ExtractStepConfigurations(jsonObject, map[string]struct{}{"first": {}, "second": {}}); err != nil {
		t.Fatalf("JSON round trip failed: %v", err)
	}
	yamlPayload, err := yaml.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var yamlObject map[any]any
	if err := yaml.Unmarshal(yamlPayload, &yamlObject); err != nil {
		t.Fatal(err)
	}
	configs, err := ExtractStepConfigurations(yamlObject, map[string]struct{}{"first": {}, "second": {}})
	if err != nil {
		t.Fatalf("YAML round trip failed: %v", err)
	}
	if !configs["first"].Enabled || configs["second"].Enabled {
		t.Fatalf("unexpected round-trip states: %#v", configs)
	}
}

func TestExtractStepConfigurationsRequiresEveryWorkflowStep(t *testing.T) {
	configs, err := ExtractCompleteStepConfigurations(map[string]any{
		"steps": map[string]any{
			"first":  map[string]any{"enabled": true, "engineConfig": map[string]any{"scan": map[string]any{"enabled": true}}},
			"second": map[string]any{"enabled": false},
		},
	}, map[string]struct{}{"first": {}, "second": {}})
	if err != nil {
		t.Fatalf("ExtractStepConfigurations failed: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("expected two complete configs, got %#v", configs)
	}

	_, err = ExtractCompleteStepConfigurations(map[string]any{
		"steps": map[string]any{"first": map[string]any{"enabled": true, "engineConfig": map[string]any{}}},
	}, map[string]struct{}{"first": {}, "second": {}})
	if err == nil || !strings.Contains(err.Error(), "configuration.steps must contain exactly every workflow step") {
		t.Fatalf("expected missing-step error, got %v", err)
	}
}

func TestDecodeReturnsDynamicFirstViolationAndStableReason(t *testing.T) {
	_, err := Decode(map[string]any{
		"steps": map[string]any{
			"port-scan": map[string]any{"engineConfig": map[string]any{}},
		},
	}, map[string]struct{}{"port-scan": {}})
	violation, ok := FirstViolation(err)
	if !ok {
		t.Fatalf("expected typed validation error, got %v", err)
	}
	if violation.Path != `configuration.steps["port-scan"].enabled` || violation.Reason != ReasonFieldRequired {
		t.Fatalf("unexpected violation: %+v", violation)
	}
}

func TestDecodeRejectsNullAndNonBooleanEnabled(t *testing.T) {
	values := []any{nil, "false", float64(0), []any{}, map[string]any{}}
	for _, value := range values {
		_, err := Decode(map[string]any{
			"steps": map[string]any{"step": map[string]any{"enabled": value}},
		}, map[string]struct{}{"step": {}})
		violation, ok := FirstViolation(err)
		if !ok || violation.Reason != ReasonFieldTypeInvalid || violation.Path != `configuration.steps["step"].enabled` {
			t.Fatalf("enabled=%#v produced %+v, %v", value, violation, err)
		}
	}
}

func TestDecodeRejectsAllDisabledRequestsButExtractKeepsProfileDraftValid(t *testing.T) {
	configuration := map[string]any{"steps": map[string]any{
		"first":  map[string]any{"enabled": false},
		"second": map[string]any{"enabled": false},
	}}
	if _, err := Decode(configuration, map[string]struct{}{"first": {}, "second": {}}); err == nil {
		t.Fatal("all-disabled request must fail")
	} else if violation, ok := FirstViolation(err); !ok || violation.Reason != ReasonFieldValueInvalid {
		t.Fatalf("unexpected all-disabled violation: %+v, %v", violation, err)
	}
	if _, err := ExtractStepConfigurations(configuration, map[string]struct{}{"first": {}, "second": {}}); err != nil {
		t.Fatalf("Profile draft shape should remain extractable: %v", err)
	}
}

func TestDecodeReturnsStableReasonsForLegacyAndUnknownEnvelopeFields(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		path   string
		reason ViolationReason
	}{
		{
			name:   "legacy workflow alias",
			value:  map[string]any{"workflow": "default", "steps": map[string]any{"step": map[string]any{"enabled": true, "engineConfig": map[string]any{}}}},
			path:   "configuration.workflow",
			reason: ReasonFieldUnknown,
		},
		{
			name:   "legacy operation config alias",
			value:  map[string]any{"steps": map[string]any{"step": map[string]any{"enabled": true, "engineConfig": map[string]any{}, "operationConfig": map[string]any{}}}},
			path:   `configuration.steps["step"].operationConfig`,
			reason: ReasonFieldUnknown,
		},
		{
			name:   "disabled unknown sibling",
			value:  map[string]any{"steps": map[string]any{"step": map[string]any{"enabled": false, "legacy": true}}},
			path:   `configuration.steps["step"].legacy`,
			reason: ReasonFieldUnknown,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Decode(test.value, map[string]struct{}{"step": {}})
			violation, ok := FirstViolation(err)
			if !ok || violation.Path != test.path || violation.Reason != test.reason {
				t.Fatalf("unexpected violation: %+v, %v", violation, err)
			}
		})
	}
}

func TestDecodePreservesTypedViolationThroughWrapping(t *testing.T) {
	_, err := Decode(map[string]any{
		"steps": map[string]any{"step": map[string]any{"enabled": nil}},
	}, map[string]struct{}{"step": {}})
	wrapped := errors.Join(errors.New("boundary"), err)
	violation, ok := FirstViolation(wrapped)
	if !ok || violation.Path != `configuration.steps["step"].enabled` || violation.Reason != ReasonFieldTypeInvalid {
		t.Fatalf("wrapped violation was not preserved: %+v, %v", violation, wrapped)
	}
}

func TestDecodeAppliesTheSameEnvelopeToBuiltinAndCustomStepIDs(t *testing.T) {
	for _, workflowID := range []string{"default", "wf-00000000-0000-4000-8000-000000000001"} {
		t.Run(workflowID, func(t *testing.T) {
			_, err := Decode(map[string]any{
				"steps": map[string]any{
					"step": map[string]any{"enabled": true, "engineConfig": map[string]any{}},
				},
			}, map[string]struct{}{"step": {}})
			if err != nil {
				t.Fatalf("%s envelope rejected: %v", workflowID, err)
			}
		})
	}
}
