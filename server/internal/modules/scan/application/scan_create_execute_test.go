package application

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type scanCreateStoreStub struct{}

func (scanCreateStoreStub) CreateWithInputTargetsAndTasks(*CreateScan, []CreateScanInputTarget, []CreateScanTask) error {
	return nil
}

type scanCreateStoreCaptureStub struct {
	lastScan  *CreateScan
	lastTasks []CreateScanTask
}

func (stub *scanCreateStoreCaptureStub) CreateWithInputTargetsAndTasks(scan *CreateScan, _ []CreateScanInputTarget, tasks []CreateScanTask) error {
	stub.lastScan = scan
	stub.lastTasks = tasks
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

func TestCreateNormal_RejectsEngineIDsAndNamesMismatch(t *testing.T) {
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
		EngineIDs:     []int{101, 102},
		EngineNames:   []string{"subdomain_discovery"},
		Configuration: validSubdomainDiscoveryConfig("v1", "1.0.0"),
	}
	_, err := service.CreateNormal(input)
	if !errors.Is(err, ErrCreateInvalidEngineNames) {
		t.Fatalf("expected invalid engine names error, got: %v", err)
	}
}

func TestCreateNormal_PersistsEngineIDsAndNamesWithSameSemanticSet(t *testing.T) {
	store := &scanCreateStoreCaptureStub{}
	service := NewScanCreateService(store, func(int) (*TargetRef, error) {
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
		EngineIDs:     []int{101},
		EngineNames:   []string{"subdomain_discovery"},
		Configuration: validSubdomainDiscoveryConfig("v1", "1.0.0"),
	}
	scan, err := service.CreateNormal(input)
	if err != nil {
		t.Fatalf("CreateNormal returned error: %v", err)
	}
	if scan == nil || store.lastScan == nil {
		t.Fatalf("expected scan persisted")
	}
	if len(store.lastScan.EngineIDs) != 1 || store.lastScan.EngineIDs[0] != 101 {
		t.Fatalf("unexpected engineIDs persisted: %v", store.lastScan.EngineIDs)
	}
	var names []string
	if err := json.Unmarshal(store.lastScan.EngineNames, &names); err != nil {
		t.Fatalf("unmarshal engineNames failed: %v", err)
	}
	if len(names) != 1 || names[0] != "subdomain_discovery" {
		t.Fatalf("unexpected engineNames persisted: %v", names)
	}
	if len(store.lastTasks) != 1 || store.lastTasks[0].Config == "" {
		t.Fatalf("expected per-task workflow config slice persisted")
	}
}

func TestCreateNormal_RejectsEngineNotInCatalog(t *testing.T) {
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
		EngineNames:   []string{"future_workflow"},
		Configuration: "future_workflow:\n  apiVersion: v1\n  schemaVersion: 1.0.0\n",
	}
	_, err := service.CreateNormal(input)
	if err == nil {
		t.Fatalf("expected engine catalog rejection")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok || workflowErr.Code != WorkflowErrorCodeSchemaInvalid {
		t.Fatalf("expected schema invalid workflow error, got: %v", err)
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
