package application

import (
	"errors"
	"testing"
)

type workflowCatalogStub struct {
	items []string
	err   error
}

func (stub workflowCatalogStub) List() ([]string, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return append([]string(nil), stub.items...), nil
}

func TestValidateEngineCatalogCoverage_UsesCatalogAdapter(t *testing.T) {
	service := &ScanCreateService{
		workflowCatalog: workflowCatalogStub{items: []string{"subdomain_discovery", "url_fetch"}},
	}
	if err := service.validateEngineCatalogCoverage([]string{"subdomain_discovery"}); err != nil {
		t.Fatalf("expected catalog coverage success, got: %v", err)
	}
}

func TestValidateEngineCatalogCoverage_RejectsUnknownWorkflow(t *testing.T) {
	service := &ScanCreateService{
		workflowCatalog: workflowCatalogStub{items: []string{"subdomain_discovery"}},
	}
	err := service.validateEngineCatalogCoverage([]string{"future_workflow"})
	if err == nil {
		t.Fatalf("expected unknown workflow rejection")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok || workflowErr.Code != WorkflowErrorCodeSchemaInvalid {
		t.Fatalf("expected schema invalid workflow error, got: %v", err)
	}
}

func TestValidateEngineCatalogCoverage_PropagatesCatalogError(t *testing.T) {
	catalogErr := errors.New("catalog unavailable")
	service := &ScanCreateService{
		workflowCatalog: workflowCatalogStub{err: catalogErr},
	}
	err := service.validateEngineCatalogCoverage([]string{"subdomain_discovery"})
	if !errors.Is(err, catalogErr) {
		t.Fatalf("expected catalog error passthrough, got: %v", err)
	}
}
