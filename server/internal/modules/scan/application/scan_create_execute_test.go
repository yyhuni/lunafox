package application

import (
	"errors"
	"testing"
	"time"
)

type scanCreateStoreStub struct{}

func (scanCreateStoreStub) CreateWithInputTargetsAndTasks(*CreateScan, []CreateScanInputTarget, []CreateScanTask) error {
	return nil
}

func TestCreateNormal_SchemaGateFailureReturnsWorkflowError(t *testing.T) {
	service := NewScanCreateService(scanCreateStoreStub{}, func(int) (*TargetRef, error) {
		now := time.Now().UTC()
		return &TargetRef{
			ID:        1,
			Name:      "example.com",
			Type:      "domain",
			CreatedAt: now,
		}, nil
	})

	input := &CreateNormalInput{
		TargetID:      1,
		EngineNames:   []string{"subdomain_discovery"},
		Configuration: "subdomain_discovery:\n  schemaVersion: 1.0.0\n",
	}
	_, err := service.CreateNormal(input)
	if err == nil {
		t.Fatalf("expected schema validation error")
	}
	if !errors.Is(err, ErrCreateInvalidConfig) {
		t.Fatalf("expected wrapped ErrCreateInvalidConfig, got: %v", err)
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok {
		t.Fatalf("expected WorkflowError, got: %v", err)
	}
	if workflowErr.Code != WorkflowErrorCodeSchemaInvalid {
		t.Fatalf("unexpected code: %s", workflowErr.Code)
	}
	if workflowErr.Stage != WorkflowErrorStageServerSchemaGate {
		t.Fatalf("unexpected stage: %s", workflowErr.Stage)
	}
	if workflowErr.Field != "subdomain_discovery" {
		t.Fatalf("unexpected field: %s", workflowErr.Field)
	}
}

func TestCreateNormal_VersionEnumRejectsUnsupportedAPIVersion(t *testing.T) {
	service := NewScanCreateService(scanCreateStoreStub{}, func(int) (*TargetRef, error) {
		now := time.Now().UTC()
		return &TargetRef{
			ID:        1,
			Name:      "example.com",
			Type:      "domain",
			CreatedAt: now,
		}, nil
	})

	input := &CreateNormalInput{
		TargetID:      1,
		EngineNames:   []string{"subdomain_discovery"},
		Configuration: validSubdomainDiscoveryConfig("v2", "1.0.0"),
	}
	_, err := service.CreateNormal(input)
	if err == nil {
		t.Fatalf("expected schema validation error")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok {
		t.Fatalf("expected WorkflowError, got: %v", err)
	}
	if workflowErr.Code != WorkflowErrorCodeSchemaInvalid {
		t.Fatalf("unexpected code: %s", workflowErr.Code)
	}
	if workflowErr.Stage != WorkflowErrorStageServerSchemaGate {
		t.Fatalf("unexpected stage: %s", workflowErr.Stage)
	}
}

func TestCreateNormal_VersionEnumRejectsUnsupportedSchemaVersion(t *testing.T) {
	service := NewScanCreateService(scanCreateStoreStub{}, func(int) (*TargetRef, error) {
		now := time.Now().UTC()
		return &TargetRef{
			ID:        1,
			Name:      "example.com",
			Type:      "domain",
			CreatedAt: now,
		}, nil
	})

	input := &CreateNormalInput{
		TargetID:      1,
		EngineNames:   []string{"subdomain_discovery"},
		Configuration: validSubdomainDiscoveryConfig("v1", "1.0.1"),
	}
	_, err := service.CreateNormal(input)
	if err == nil {
		t.Fatalf("expected schema validation error")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok {
		t.Fatalf("expected WorkflowError, got: %v", err)
	}
	if workflowErr.Code != WorkflowErrorCodeSchemaInvalid {
		t.Fatalf("unexpected code: %s", workflowErr.Code)
	}
	if workflowErr.Stage != WorkflowErrorStageServerSchemaGate {
		t.Fatalf("unexpected stage: %s", workflowErr.Stage)
	}
}

func validSubdomainDiscoveryConfig(apiVersion, schemaVersion string) string {
	return "subdomain_discovery:\n" +
		"  apiVersion: " + apiVersion + "\n" +
		"  schemaVersion: " + schemaVersion + "\n" +
		"  recon:\n" +
		"    enabled: false\n" +
		"    tools:\n" +
		"      subfinder:\n" +
		"        enabled: false\n" +
		"  bruteforce:\n" +
		"    enabled: false\n" +
		"    tools:\n" +
		"      subdomain-bruteforce:\n" +
		"        enabled: false\n" +
		"  permutation:\n" +
		"    enabled: false\n" +
		"    tools:\n" +
		"      subdomain-permutation-resolve:\n" +
		"        enabled: false\n" +
		"  resolve:\n" +
		"    enabled: false\n" +
		"    tools:\n" +
		"      subdomain-resolve:\n" +
		"        enabled: false\n"
}
