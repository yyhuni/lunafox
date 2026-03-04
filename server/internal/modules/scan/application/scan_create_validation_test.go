package application

import "testing"

func TestValidateRequestedEngines_KnownWorkflow(t *testing.T) {
	service := &ScanCreateService{}
	if err := service.validateRequestedEngines([]string{"subdomain_discovery"}); err != nil {
		t.Fatalf("expected known workflow coverage success, got: %v", err)
	}
}

func TestValidateRequestedEngines_RejectsUnknownWorkflow(t *testing.T) {
	service := &ScanCreateService{}
	err := service.validateRequestedEngines([]string{"future_workflow"})
	if err == nil {
		t.Fatalf("expected unknown workflow rejection")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok || workflowErr.Code != WorkflowErrorCodeSchemaInvalid {
		t.Fatalf("expected schema invalid workflow error, got: %v", err)
	}
}
