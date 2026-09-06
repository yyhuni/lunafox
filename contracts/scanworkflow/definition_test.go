package scanworkflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateComponentIDUsesDefinitionGrammar(t *testing.T) {
	for _, value := range []string{"discovery", "port_scan", "web-sites", "a1"} {
		if err := ValidateComponentID(value); err != nil {
			t.Fatalf("ValidateComponentID(%q) error = %v", value, err)
		}
	}
	for _, value := range []string{"", "1bad", " bad", "Bad", "bad.step", strings.Repeat("a", 129)} {
		if err := ValidateComponentID(value); err == nil {
			t.Fatalf("ValidateComponentID(%q) succeeded", value)
		}
	}
}

func TestDecodeDefinitionRejectsUnknownFields(t *testing.T) {
	payload := []byte(`{"scanWorkflowId":"subdomain_discovery","displayName":"Subdomain Discovery","description":"Discover subdomains.","stages":[{"stageId":"discovery","steps":[{"stepId":"subdomain_discovery","engineId":"engine.lunafox.subdomain_discovery","profileDefaultEnabled":true}]}],"executor":{"ref":"runtime.subdomain_discovery"}}`)

	_, err := DecodeDefinition(payload, "test.scan-workflow.json")

	if err == nil || !strings.Contains(err.Error(), "executor") {
		t.Fatalf("expected unknown field rejection, got %v", err)
	}
}

func TestValidateDefinitionRejectsDuplicateStepAcrossStages(t *testing.T) {
	definition := Definition{
		ScanWorkflowID: "subdomain_discovery",
		DisplayName:    "Subdomain Discovery",
		Description:    "Discover subdomains.",
		Stages: []Stage{
			{StageID: "first", Steps: []Step{{StepID: "subdomain_discovery", EngineID: "engine.lunafox.subdomain_discovery", ProfileDefaultEnabled: true}}},
			{StageID: "second", Steps: []Step{{StepID: "subdomain_discovery", EngineID: "engine.lunafox.subdomain_discovery", ProfileDefaultEnabled: true}}},
		},
	}

	err := ValidateDefinition(definition)

	if err == nil || !strings.Contains(err.Error(), "duplicate stepId") {
		t.Fatalf("expected duplicate step rejection, got %v", err)
	}
}

func TestDecodeDefinitionRejectsStepEngineConfig(t *testing.T) {
	payload := []byte(`{"scanWorkflowId":"subdomain_discovery","displayName":"Subdomain Discovery","description":"Discover subdomains.","stages":[{"stageId":"discovery","steps":[{"stepId":"subdomain_discovery","engineId":"engine.lunafox.subdomain_discovery","profileDefaultEnabled":true,"engineConfig":{"recon":{"enabled":true}}}]}]}`)

	_, err := DecodeDefinition(payload, "test.scan-workflow.json")
	if err == nil || !strings.Contains(err.Error(), "engineConfig") {
		t.Fatalf("expected engineConfig rejection, got %v", err)
	}
}

func TestDecodeDefinitionRejectsWorkflowStepConfigurationAndExecutionSelectors(t *testing.T) {
	for _, field := range []string{"enabled", "defaultEnabled", "packageDigest", "operationId"} {
		payload := []byte(`{"scanWorkflowId":"subdomain_discovery","displayName":"Subdomain Discovery","description":"Discover subdomains.","stages":[{"stageId":"discovery","steps":[{"stepId":"subdomain_discovery","engineId":"engine.lunafox.subdomain_discovery","profileDefaultEnabled":true,"` + field + `":true}]}]}`)
		_, err := DecodeDefinition(payload, "test.scan-workflow.json")
		if err == nil || !strings.Contains(err.Error(), field) {
			t.Fatalf("expected pure-topology rejection for %s, got %v", field, err)
		}
	}
}

func TestDecodeDefinitionKeepsBuiltinAndCustomWorkflowStepsPureTopology(t *testing.T) {
	for _, workflowID := range []string{"default", "wf-00000000-0000-4000-8000-000000000001"} {
		t.Run(workflowID, func(t *testing.T) {
			payload := []byte(`{"scanWorkflowId":"` + workflowID + `","displayName":"Discovery","description":"Discover assets.","stages":[{"stageId":"discovery","steps":[{"stepId":"discover","engineId":"engine.lunafox.discovery","profileDefaultEnabled":true}]}]}`)
			definition, err := DecodeDefinition(payload, "test.scan-workflow.json")
			if err != nil {
				t.Fatalf("pure topology decode failed: %v", err)
			}
			if len(definition.Stages) != 1 || len(definition.Stages[0].Steps) != 1 || definition.Stages[0].Steps[0].EngineID != "engine.lunafox.discovery" {
				t.Fatalf("unexpected topology: %+v", definition)
			}
		})
	}
}

func TestDecodeDefinitionRequiresProfileDefaultEnabledBoolean(t *testing.T) {
	for name, value := range map[string]string{
		"missing": "",
		"null":    `,"profileDefaultEnabled":null`,
		"string":  `,"profileDefaultEnabled":"true"`,
		"number":  `,"profileDefaultEnabled":1`,
	} {
		t.Run(name, func(t *testing.T) {
			payload := []byte(`{"scanWorkflowId":"default","displayName":"Default","description":"Run discovery.","stages":[{"stageId":"discovery","steps":[{"stepId":"discover","engineId":"engine.lunafox.discovery"` + value + `}]}]}`)
			_, err := DecodeDefinition(payload, "test.scan-workflow.json")
			if err == nil || !strings.Contains(err.Error(), "profileDefaultEnabled") {
				t.Fatalf("expected strict profileDefaultEnabled rejection, got %v", err)
			}
		})
	}
}

func TestDecodeDefinitionPreservesExplicitProfileDefaultEnabledValues(t *testing.T) {
	payload := []byte(`{"scanWorkflowId":"default","displayName":"Default","description":"Run discovery.","stages":[{"stageId":"discovery","steps":[{"stepId":"enabled","engineId":"engine.lunafox.discovery","profileDefaultEnabled":true},{"stepId":"disabled","engineId":"engine.lunafox.other","profileDefaultEnabled":false}]}]}`)
	definition, err := DecodeDefinition(payload, "test.scan-workflow.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := definition.Stages[0].Steps; !got[0].ProfileDefaultEnabled || got[1].ProfileDefaultEnabled {
		t.Fatalf("explicit Profile defaults were not preserved: %+v", got)
	}
	encoded, err := json.Marshal(definition.Stages[0].Steps)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"profileDefaultEnabled":true`) || !strings.Contains(string(encoded), `"profileDefaultEnabled":false`) {
		t.Fatalf("encoded steps omitted explicit Profile defaults: %s", encoded)
	}
}

func TestValidateDefinitionAllowsEngineOnlyStep(t *testing.T) {
	definition := Definition{
		ScanWorkflowID: "default",
		DisplayName:    "Subdomain Discovery",
		Description:    "Discover subdomains.",
		Stages: []Stage{
			{StageID: "discovery", Steps: []Step{{StepID: "discover_subdomains", EngineID: "engine.lunafox.subdomain_discovery", ProfileDefaultEnabled: true}}},
		},
	}

	err := ValidateDefinition(definition)

	if err != nil {
		t.Fatalf("expected engine-only step to pass, got %v", err)
	}
}

func TestValidateDefinitionAllowsDefaultWorkflowID(t *testing.T) {
	definition := Definition{
		ScanWorkflowID: "default",
		DisplayName:    "Default Scan",
		Description:    "Run the default builtin scan workflow.",
		Stages: []Stage{
			{StageID: "discovery", Steps: []Step{{StepID: "subdomain_discovery", EngineID: "engine.lunafox.subdomain_discovery", ProfileDefaultEnabled: true}}},
		},
	}

	if err := ValidateDefinition(definition); err != nil {
		t.Fatalf("expected default workflow ID to pass, got %v", err)
	}
}

func TestDecodeDefinitionRejectsLegacyWorkflowIDField(t *testing.T) {
	payload := []byte(`{"workflowId":"subdomain_discovery","displayName":"Subdomain Discovery","description":"Discover subdomains.","stages":[{"stageId":"discovery","steps":[{"stepId":"subdomain_discovery","engineId":"engine.lunafox.subdomain_discovery","profileDefaultEnabled":true}]}]}`)

	_, err := DecodeDefinition(payload, "test.scan-workflow.json")

	if err == nil || !strings.Contains(err.Error(), "workflowId") {
		t.Fatalf("expected legacy workflowId rejection, got %v", err)
	}
}

func TestDecodeDefinitionRejectsLegacyOperationIDField(t *testing.T) {
	payload := []byte(`{"scanWorkflowId":"subdomain_discovery","displayName":"Subdomain Discovery","description":"Discover subdomains.","stages":[{"stageId":"discovery","steps":[{"stepId":"subdomain_discovery","engineId":"engine.lunafox.subdomain_discovery","profileDefaultEnabled":true,"operationId":"subdomain_discovery"}]}]}`)

	_, err := DecodeDefinition(payload, "test.scan-workflow.json")

	if err == nil || !strings.Contains(err.Error(), "operationId") {
		t.Fatalf("expected operationId rejection, got %v", err)
	}
}

func TestDecodeDefinitionRejectsLegacyOperationConfigField(t *testing.T) {
	payload := []byte(`{"scanWorkflowId":"subdomain_discovery","displayName":"Subdomain Discovery","description":"Discover subdomains.","stages":[{"stageId":"discovery","steps":[{"stepId":"subdomain_discovery","engineId":"engine.lunafox.subdomain_discovery","profileDefaultEnabled":true,"operationConfig":{"recon":{"enabled":true}}}]}]}`)

	_, err := DecodeDefinition(payload, "test.scan-workflow.json")

	if err == nil || !strings.Contains(err.Error(), "operationConfig") {
		t.Fatalf("expected operationConfig rejection, got %v", err)
	}
}
