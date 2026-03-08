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
		return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
	})

	input := &CreateNormalInput{
		TargetID:    1,
		WorkflowIDs: []string{"subdomain_discovery"},
		Configuration: map[string]any{
			"subdomain_discovery": map[string]any{
				"recon": map[string]any{"enabled": true},
			},
		},
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
		return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
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
		return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
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
		return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
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
		return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
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
	if store.lastScan.Configuration == nil {
		t.Fatalf("expected canonical scan configuration object persisted")
	}
	if len(store.lastTasks) != 1 || store.lastTasks[0].WorkflowConfig == nil {
		t.Fatalf("expected per-task workflow config object persisted")
	}
}

func TestCreateNormal_RejectsWorkflowNotInCatalog(t *testing.T) {
	service := NewScanCreateService(scanCreateStoreStub{}, func(int) (*TargetRef, error) {
		now := time.Now().UTC()
		return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
	})

	input := &CreateNormalInput{
		TargetID:    1,
		WorkflowIDs: []string{"future_workflow"},
		Configuration: map[string]any{
			"future_workflow": map[string]any{"enabled": true},
		},
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

	workflowRoot, ok := store.lastScan.Configuration["subdomain_discovery"].(map[string]any)
	if !ok {
		t.Fatalf("expected canonical scan object to include subdomain_discovery config, got: %#v", store.lastScan.Configuration)
	}
	threads := nestedInt(workflowRoot, "recon", "tools", "subfinder", "threads-cli")
	if threads != 10 {
		t.Fatalf("expected default threads-cli 10 in canonical object, got %d", threads)
	}
	if len(store.lastTasks) != 1 {
		t.Fatalf("expected one task, got %d", len(store.lastTasks))
	}
	threads = nestedInt(store.lastTasks[0].WorkflowConfig, "recon", "tools", "subfinder", "threads-cli")
	if threads != 10 {
		t.Fatalf("expected task workflow object to include recon default threads-cli, got %d", threads)
	}
}

func validSubdomainDiscoveryConfig() map[string]any {
	return map[string]any{
		"subdomain_discovery": map[string]any{
			"recon": map[string]any{
				"enabled": false,
				"tools": map[string]any{
					"subfinder": map[string]any{
						"enabled": false,
					},
				},
			},
			"bruteforce": map[string]any{
				"enabled": false,
				"tools": map[string]any{
					"subdomain-bruteforce": map[string]any{
						"enabled": false,
					},
				},
			},
			"permutation": map[string]any{
				"enabled": false,
				"tools": map[string]any{
					"subdomain-permutation-resolve": map[string]any{
						"enabled": false,
					},
				},
			},
			"resolve": map[string]any{
				"enabled": false,
				"tools": map[string]any{
					"subdomain-resolve": map[string]any{
						"enabled": false,
					},
				},
			},
		},
	}
}

func shortSubdomainDiscoveryConfig() map[string]any {
	return map[string]any{
		"subdomain_discovery": map[string]any{
			"recon": map[string]any{
				"enabled": true,
				"tools": map[string]any{
					"subfinder": map[string]any{
						"enabled": true,
					},
				},
			},
			"bruteforce": map[string]any{
				"enabled": false,
				"tools": map[string]any{
					"subdomain-bruteforce": map[string]any{
						"enabled": false,
					},
				},
			},
			"permutation": map[string]any{
				"enabled": false,
				"tools": map[string]any{
					"subdomain-permutation-resolve": map[string]any{
						"enabled": false,
					},
				},
			},
			"resolve": map[string]any{
				"enabled": false,
				"tools": map[string]any{
					"subdomain-resolve": map[string]any{
						"enabled": false,
					},
				},
			},
		},
	}
}

func nestedInt(root map[string]any, path ...string) int {
	current := any(root)
	for _, key := range path {
		mapping, ok := current.(map[string]any)
		if !ok {
			return 0
		}
		current = mapping[key]
	}
	switch value := current.(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}
