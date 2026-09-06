package application

import (
	"context"
	"io/fs"
	"testing"
)

func TestCloneMapDeepCopiesNestedConfigurationValues(t *testing.T) {
	input := map[string]any{
		"nested": map[string]any{
			"items": []any{map[string]any{"value": 1}},
		},
		"typed": []int{1, 2},
	}
	cloned := cloneMap(input)

	input["nested"].(map[string]any)["items"].([]any)[0].(map[string]any)["value"] = 99
	input["typed"].([]int)[0] = 42

	if got := cloned["nested"].(map[string]any)["items"].([]any)[0].(map[string]any)["value"]; got != 1 {
		t.Fatalf("nested configuration was not cloned: %#v", cloned)
	}
	if got := cloned["typed"].([]int)[0]; got != 1 {
		t.Fatalf("typed slice was not cloned: %#v", cloned)
	}
}

func TestNormalizePlanTaskConfigurationRejectsUnknownScanWorkflowStep(t *testing.T) {
	manifest := ScanCreateWorkflowManifest{
		ScanWorkflowID: "subdomain_discovery",
		Stages: []ScanCreateWorkflowStage{{
			StageID: "discovery",
			Steps: []ScanCreateWorkflowStep{{
				StepID:   "subdomain_discovery",
				EngineID: "engine.lunafox.subdomain_discovery",
			}},
		}},
	}

	_, err := normalizePlanTaskConfiguration(map[string]any{
		"steps": map[string]any{
			"unknown": map[string]any{
				"enabled": false,
			},
		},
	}, manifest)
	if err == nil {
		t.Fatal("expected unknown scan workflow step to fail fast")
	}
}

func TestNormalizePlanTaskConfigurationRejectsMissingWorkflowStep(t *testing.T) {
	manifest := ScanCreateWorkflowManifest{
		ScanWorkflowID: "subdomain_discovery",
		Stages: []ScanCreateWorkflowStage{{
			StageID: "discovery",
			Steps: []ScanCreateWorkflowStep{{
				StepID:   "subdomain_discovery",
				EngineID: "engine.lunafox.subdomain_discovery",
			}},
		}},
	}

	_, err := normalizePlanTaskConfiguration(map[string]any{
		"steps": map[string]any{},
	}, manifest)
	if err == nil {
		t.Fatal("expected missing workflow step to fail fast")
	}
}

func TestValidateRequestedScanWorkflowAcceptsCanonicalScanWorkflowRef(t *testing.T) {
	workflowReader, _ := scanCreateReaderStubs()
	service := &ScanCreateService{workflowReader: workflowReader}
	manifest, err := service.validateRequestedScanWorkflow(context.Background(), "scanWorkflows/default")
	if err != nil {
		t.Fatalf("expected known scan workflow coverage success, got: %v", err)
	}
	if manifest.ScanWorkflowID != "subdomain_discovery" {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
}

func TestValidateRequestedScanWorkflowRejectsTypedRefAtBoundary(t *testing.T) {
	workflowReader, _ := scanCreateReaderStubs()
	service := &ScanCreateService{workflowReader: workflowReader}
	for _, workflow := range []string{
		"engine.lunafox.subdomain_discovery",
		"runtime.subdomain_discovery",
		"schema.engine.subdomain_discovery",
		"workflow.subdomain_discovery",
		"scanWorkflows/engine.lunafox.subdomain_discovery",
	} {
		if _, err := service.validateRequestedScanWorkflow(context.Background(), workflow); err == nil {
			t.Fatalf("expected typed ref rejection for %q", workflow)
		}
	}
}

func TestValidateRequestedScanWorkflowUsesCanonicalResourceNameParser(t *testing.T) {
	workflowReader, _ := scanCreateReaderStubs()
	service := &ScanCreateService{workflowReader: workflowReader}
	for _, workflow := range []string{
		"scanWorkflows/Subdomain_Discovery",
		"scanWorkflows/subdomain-discovery",
		"scanWorkflows/_subdomain_discovery",
		"scanWorkflows/subdomain-discovery",
	} {
		if _, err := service.validateRequestedScanWorkflow(context.Background(), workflow); err == nil {
			t.Fatalf("expected contract resource name rejection for %q", workflow)
		}
	}
}

func TestValidateRequestedScanWorkflowRejectsMissingFormalWorkflowDefinition(t *testing.T) {
	service := &ScanCreateService{workflowReader: scanCreateWorkflowReaderStub{err: fs.ErrNotExist}}
	if _, err := service.validateRequestedScanWorkflow(context.Background(), "scanWorkflows/unknown_workflow"); err == nil {
		t.Fatal("expected missing scan workflow definition rejection")
	}
}
