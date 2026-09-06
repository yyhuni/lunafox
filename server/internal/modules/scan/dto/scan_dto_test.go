package dto

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBatchCreateScanRequestRejectsWorkflowIDsField(t *testing.T) {
	payload := []byte(`{"requests":[{"target":"targets/1"}],"workflowIds":["subdomain_discovery"],"configuration":{}}`)
	var request BatchCreateScanRequest

	err := json.Unmarshal(payload, &request)

	if err == nil || !strings.Contains(err.Error(), "workflowIds") {
		t.Fatalf("expected workflowIds rejection, got %v", err)
	}
}

func TestBatchCreateScanRequestRejectsInternalWorkflowIDField(t *testing.T) {
	payload := []byte(`{"requests":[{"target":"targets/1"}],"workflowId":"subdomain_discovery","configuration":{}}`)
	var request BatchCreateScanRequest

	err := json.Unmarshal(payload, &request)

	if err == nil || !strings.Contains(err.Error(), "workflowId") {
		t.Fatalf("expected workflowId rejection, got %v", err)
	}
}

func TestBatchCreateScanRequestRejectsLegacyWorkflowField(t *testing.T) {
	payload := []byte(`{"requests":[{"target":"targets/1"}],"workflow":"scanWorkflows/default","configuration":{"steps":{}}}`)
	var request BatchCreateScanRequest

	err := json.Unmarshal(payload, &request)

	if err == nil || !strings.Contains(err.Error(), "workflow") {
		t.Fatalf("expected workflow rejection, got %v", err)
	}
}

func TestBatchCreateScanRequestRejectsDirectEnginesField(t *testing.T) {
	payload := []byte(`{"requests":[{"target":"targets/1"}],"engines":["engine.lunafox.subdomain_discovery"],"configuration":{"steps":{}}}`)
	var request BatchCreateScanRequest

	err := json.Unmarshal(payload, &request)

	if err == nil || !strings.Contains(err.Error(), "engines") || !strings.Contains(err.Error(), "scanWorkflow") {
		t.Fatalf("expected direct engines rejection with scanWorkflow guidance, got %v", err)
	}
}

func TestBatchCreateScanRequestAcceptsCanonicalScanWorkflowField(t *testing.T) {
	payload := []byte(`{"requests":[{"target":"targets/1"}],"scanWorkflow":"scanWorkflows/default","configuration":{"steps":{}}}`)
	var request BatchCreateScanRequest

	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatalf("expected canonical request to decode, got %v", err)
	}
	if request.ScanWorkflow != "scanWorkflows/default" || len(request.Requests) != 1 || request.Requests[0].Target != "targets/1" {
		t.Fatalf("unexpected scan workflow: %+v", request)
	}
}

func TestBatchCreateScanItemRejectsUnknownScopeField(t *testing.T) {
	payload := []byte(`{"source":"targets/1"}`)
	var request BatchCreateScanItem

	err := json.Unmarshal(payload, &request)

	if err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("expected source rejection, got %v", err)
	}
}

func TestCreateQuickScanRequestRejectsLegacyWorkflowField(t *testing.T) {
	payload := []byte(`{"targets":["example.com"],"workflow":"scanWorkflows/default","configuration":{"steps":{}}}`)
	var request CreateQuickScanRequest

	err := json.Unmarshal(payload, &request)

	if err == nil || !strings.Contains(err.Error(), "workflow") {
		t.Fatalf("expected workflow rejection, got %v", err)
	}
}

func TestCreateQuickScanRequestRejectsDirectEnginesField(t *testing.T) {
	payload := []byte(`{"targets":["example.com"],"engines":["engine.lunafox.subdomain_discovery"],"configuration":{"steps":{}}}`)
	var request CreateQuickScanRequest

	err := json.Unmarshal(payload, &request)

	if err == nil || !strings.Contains(err.Error(), "engines") || !strings.Contains(err.Error(), "scanWorkflow") {
		t.Fatalf("expected direct engines rejection with scanWorkflow guidance, got %v", err)
	}
}

func TestCreateQuickScanRequestAcceptsCanonicalScanWorkflowField(t *testing.T) {
	payload := []byte(`{"targets":["example.com"],"scanWorkflow":"scanWorkflows/default","configuration":{"steps":{}}}`)
	var request CreateQuickScanRequest

	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatalf("expected canonical quick request to decode, got %v", err)
	}
	if request.ScanWorkflow != "scanWorkflows/default" || len(request.Targets) != 1 {
		t.Fatalf("unexpected quick request: %+v", request)
	}
}

func TestPublicCreateRequestsRejectCallerSuppliedTriggerType(t *testing.T) {
	tests := []struct {
		name   string
		decode func() error
	}{
		{
			name: "quick",
			decode: func() error {
				var request CreateQuickScanRequest
				return json.Unmarshal([]byte(`{"targets":["example.com"],"scanWorkflow":"scanWorkflows/default","configuration":{},"triggerType":"scheduled"}`), &request)
			},
		},
		{
			name: "batch",
			decode: func() error {
				var request BatchCreateScanRequest
				return json.Unmarshal([]byte(`{"requests":[{"target":"targets/1"}],"scanWorkflow":"scanWorkflows/default","configuration":{},"triggerType":"ai"}`), &request)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.decode(); err == nil || !strings.Contains(err.Error(), "triggerType") {
				t.Fatalf("expected triggerType rejection, got %v", err)
			}
		})
	}
}
