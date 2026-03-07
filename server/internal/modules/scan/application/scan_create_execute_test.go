package application

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

type scanCreateStoreStub struct{}

func (scanCreateStoreStub) CreateWithTasks(*CreateScan, []CreateScanTask) error {
	return nil
}

type scanCreateStoreCaptureStub struct {
	lastScan  *CreateScan
	lastTasks []CreateScanTask
}

func (stub *scanCreateStoreCaptureStub) CreateWithTasks(scan *CreateScan, tasks []CreateScanTask) error {
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
		WorkflowIDs:   []string{"subdomain_discovery"},
		Configuration: "subdomain_discovery:\n  recon:\n    enabled: true\n",
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

func TestCreateNormal_RejectsEmptyWorkflowID(t *testing.T) {
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
		WorkflowIDs:   []string{""},
		Configuration: validSubdomainDiscoveryConfig(),
	}
	_, err := service.CreateNormal(input)
	if !errors.Is(err, ErrCreateInvalidWorkflowIDs) {
		t.Fatalf("expected invalid workflow names error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "workflowIds[0] must not be empty") {
		t.Fatalf("expected detailed empty reason, got: %v", err)
	}
}

func TestCreateNormal_RejectsWorkflowIDWithSurroundingSpaces(t *testing.T) {
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
		WorkflowIDs:   []string{" subdomain_discovery "},
		Configuration: validSubdomainDiscoveryConfig(),
	}
	_, err := service.CreateNormal(input)
	if !errors.Is(err, ErrCreateInvalidWorkflowIDs) {
		t.Fatalf("expected invalid workflow names error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "must not contain leading or trailing spaces") {
		t.Fatalf("expected detailed spacing reason, got: %v", err)
	}
}

func TestCreateNormal_RejectsDuplicateWorkflowIDs(t *testing.T) {
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
		WorkflowIDs:   []string{"subdomain_discovery", "subdomain_discovery"},
		Configuration: validSubdomainDiscoveryConfig(),
	}
	_, err := service.CreateNormal(input)
	if !errors.Is(err, ErrCreateInvalidWorkflowIDs) {
		t.Fatalf("expected invalid workflow names error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "is duplicated") {
		t.Fatalf("expected detailed duplicate reason, got: %v", err)
	}
}

func TestCreateNormal_PersistsWorkflowIDsAndTaskConfig(t *testing.T) {
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
		WorkflowIDs:   []string{"subdomain_discovery"},
		Configuration: validSubdomainDiscoveryConfig(),
	}
	scan, err := service.CreateNormal(input)
	if err != nil {
		t.Fatalf("CreateNormal returned error: %v", err)
	}
	if scan == nil || store.lastScan == nil {
		t.Fatalf("expected scan persisted")
	}
	var names []string
	if err := json.Unmarshal(store.lastScan.WorkflowIDs, &names); err != nil {
		t.Fatalf("unmarshal workflowIds failed: %v", err)
	}
	if len(names) != 1 || names[0] != "subdomain_discovery" {
		t.Fatalf("unexpected workflowIds persisted: %v", names)
	}
	if len(store.lastTasks) != 1 || store.lastTasks[0].WorkflowConfigYAML == "" {
		t.Fatalf("expected per-task workflow config slice persisted")
	}
}

func TestCreateNormal_RejectsWorkflowNotInCatalog(t *testing.T) {
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
		WorkflowIDs:   []string{"future_workflow"},
		Configuration: "future_workflow:\n  enabled: true\n",
	}
	_, err := service.CreateNormal(input)
	if err == nil {
		t.Fatalf("expected workflow catalog rejection")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok || workflowErr.Code != WorkflowErrorCodeSchemaInvalid {
		t.Fatalf("expected schema invalid workflow error, got: %v", err)
	}
}

func TestCreateNormal_PersistsCanonicalWorkflowYAMLWithDefaults(t *testing.T) {
	store := &scanCreateStoreCaptureStub{}
	service := NewScanCreateService(store, func(int) (*TargetRef, error) {
		now := time.Now().UTC()
		return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
	})

	input := &CreateNormalInput{
		TargetID:      1,
		WorkflowIDs:   []string{"subdomain_discovery"},
		Configuration: shortSubdomainDiscoveryConfig(),
	}
	scan, err := service.CreateNormal(input)
	if err != nil {
		t.Fatalf("CreateNormal returned error: %v", err)
	}
	if scan == nil || store.lastScan == nil {
		t.Fatalf("expected scan persisted")
	}
	if !strings.Contains(store.lastScan.YAMLConfiguration, "threads-cli: 10") {
		t.Fatalf("expected canonical scan YAML to include recon default threads-cli, got: %s", store.lastScan.YAMLConfiguration)
	}
	if !strings.Contains(store.lastScan.YAMLConfiguration, "timeout-runtime: 3600") {
		t.Fatalf("expected canonical scan YAML to include recon default timeout-runtime, got: %s", store.lastScan.YAMLConfiguration)
	}
	if len(store.lastTasks) != 1 {
		t.Fatalf("expected one task, got %d", len(store.lastTasks))
	}
	if !strings.Contains(store.lastTasks[0].WorkflowConfigYAML, "threads-cli: 10") {
		t.Fatalf("expected task workflow slice to include recon default threads-cli, got: %s", store.lastTasks[0].WorkflowConfigYAML)
	}
}

func validSubdomainDiscoveryConfig() string {
	return "subdomain_discovery:\n" +
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

func shortSubdomainDiscoveryConfig() string {
	return `subdomain_discovery:
  recon:
    enabled: true
    tools:
      subfinder:
        enabled: true
  bruteforce:
    enabled: false
    tools:
      subdomain-bruteforce:
        enabled: false
  permutation:
    enabled: false
    tools:
      subdomain-permutation-resolve:
        enabled: false
  resolve:
    enabled: false
    tools:
      subdomain-resolve:
        enabled: false`
}
