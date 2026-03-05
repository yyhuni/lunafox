package application

import "testing"

func TestValidateRequestedWorkflows_KnownWorkflow(t *testing.T) {
	service := &ScanCreateService{}
	if err := service.validateRequestedWorkflows([]string{"subdomain_discovery"}); err != nil {
		t.Fatalf("expected known workflow coverage success, got: %v", err)
	}
}

func TestValidateRequestedWorkflows_RejectsUnknownWorkflow(t *testing.T) {
	service := &ScanCreateService{}
	err := service.validateRequestedWorkflows([]string{"future_workflow"})
	if err == nil {
		t.Fatalf("expected unknown workflow rejection")
	}
	workflowErr, ok := AsWorkflowError(err)
	if !ok || workflowErr.Code != WorkflowErrorCodeSchemaInvalid {
		t.Fatalf("expected schema invalid workflow error, got: %v", err)
	}
}
