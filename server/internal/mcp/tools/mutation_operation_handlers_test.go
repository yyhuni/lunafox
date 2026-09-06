package tools

import (
	"context"
	"testing"
	"time"
)

type testVulnerabilityAction struct {
	single []VulnerabilityActionInput
	batch  []VulnerabilityBatchActionInput
	err    error
}

func (action *testVulnerabilityAction) SetReviewed(_ context.Context, input VulnerabilityActionInput) (VulnerabilityActionRecord, error) {
	action.single = append(action.single, input)
	if action.err != nil {
		return VulnerabilityActionRecord{}, action.err
	}
	return VulnerabilityActionRecord{Name: input.Name, Reviewed: input.Reviewed}, nil
}

func (action *testVulnerabilityAction) BatchSetReviewed(_ context.Context, input VulnerabilityBatchActionInput) (VulnerabilityBatchActionRecord, error) {
	action.batch = append(action.batch, input)
	if action.err != nil {
		return VulnerabilityBatchActionRecord{}, action.err
	}
	return VulnerabilityBatchActionRecord{RequestedCount: len(input.IDs), ProcessedCount: len(input.IDs), Reviewed: input.Reviewed}, nil
}

type testScanStarter struct {
	input StartScanInput
	err   error
}

func (starter *testScanStarter) Start(_ context.Context, input StartScanInput) (StartScanOutput, error) {
	starter.input = input
	if starter.err != nil {
		return StartScanOutput{}, starter.err
	}
	return StartScanOutput{Operation: "operations/11111111-1111-1111-1111-111111111111", Scan: "scans/9"}, nil
}

type testOperationReader struct {
	name string
	err  error
}

func (reader *testOperationReader) GetOperation(_ context.Context, name string) (OperationRecord, error) {
	reader.name = name
	if reader.err != nil {
		return OperationRecord{}, reader.err
	}
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	return OperationRecord{Name: name, Scan: "scans/9", Target: "targets/1", Status: "SUCCEEDED", Progress: 100, CreateTime: now, UpdateTime: now, Response: map[string]any{"scan": "scans/9"}}, nil
}

func TestVulnerabilityDispositionHandlersExposeOnlyExplicitActionInputs(t *testing.T) {
	action := &testVulnerabilityAction{}
	registry := NewRegistry(Dependencies{VulnerabilityActions: action})

	result, err := registry.reviewVulnerability(context.Background(), toolRequest(`{"name":"vulnerabilities/7","request_id":"one"}`))
	if err != nil || result == nil || result.IsError || len(action.single) != 1 {
		t.Fatalf("review handler = %#v, %v, inputs=%+v", result, err, action.single)
	}
	if got := action.single[0]; got.ID != 7 || got.Action != ToolReviewVulnerability || !got.Reviewed || got.RequestID != "one" {
		t.Fatalf("review action input = %+v", got)
	}

	result, err = registry.batchUnreviewVulnerabilities(context.Background(), toolRequest(`{"names":["vulnerabilities/7","vulnerabilities/8"],"request_id":"two"}`))
	if err != nil || result == nil || result.IsError || len(action.batch) != 1 {
		t.Fatalf("batch unreview handler = %#v, %v, inputs=%+v", result, err, action.batch)
	}
	if got := action.batch[0]; got.Action != ToolBatchUnreviewVulnerabilities || got.Reviewed || len(got.IDs) != 2 || got.IDs[0] != 7 || got.IDs[1] != 8 {
		t.Fatalf("batch action input = %+v", got)
	}

	invalid, err := registry.batchReviewVulnerabilities(context.Background(), toolRequest(`{"names":["vulnerabilities/7","vulnerabilities/7"]}`))
	if err != nil || invalid == nil || !invalid.IsError || len(action.batch) != 1 {
		t.Fatalf("duplicate batch = %#v, %v, inputs=%+v", invalid, err, action.batch)
	}
	structured := invalid.StructuredContent.(map[string]any)
	if structured["error"].(map[string]any)["category"] != "INVALID_ARGUMENT" {
		t.Fatalf("duplicate batch diagnostics = %#v", structured)
	}
}

func TestScanOperationHandlersUseClosedSchemasAndCanonicalReferences(t *testing.T) {
	starter := &testScanStarter{}
	operations := &testOperationReader{}
	registry := NewRegistry(Dependencies{ScanStarter: starter, Operations: operations})

	result, err := registry.startScan(context.Background(), toolRequest(`{
		"target":"targets/1",
		"scan_workflow":"scanWorkflows/default",
		"configuration":{"steps":{"discover":{"enabled":true}}},
		"agent":"agents/2",
		"request_id":"start-one"
	}`))
	if err != nil || result == nil || result.IsError {
		t.Fatalf("start_scan = %#v, %v", result, err)
	}
	if got := starter.input; got.Target != "targets/1" || got.ScanWorkflow != "scanWorkflows/default" || got.Agent != "agents/2" || got.RequestID != "start-one" || got.Configuration == nil {
		t.Fatalf("start input = %+v", got)
	}
	if _, err := registry.startScan(context.Background(), toolRequest(`{"target":"targets/1","scan_workflow":"scanWorkflows/default","configuration":{"steps":{}},"package":"server-owned"}`)); err == nil {
		t.Fatal("start_scan accepted an excluded server-owned field")
	}

	operationName := "operations/11111111-1111-1111-1111-111111111111"
	result, err = registry.getOperation(context.Background(), toolRequest(`{"name":"`+operationName+`"}`))
	if err != nil || result == nil || result.IsError || operations.name != operationName {
		t.Fatalf("get_operation = %#v, %v, name=%q", result, err, operations.name)
	}

	startSchema := startScanSchema()
	if startSchema["additionalProperties"] != false {
		t.Fatalf("start_scan schema is not closed: %#v", startSchema)
	}
	properties := startSchema["properties"].(map[string]any)
	for _, excluded := range []string{"package", "plan", "secret", "command"} {
		if _, exists := properties[excluded]; exists {
			t.Fatalf("start_scan exposed server-owned %q", excluded)
		}
	}
	steps := properties["configuration"].(map[string]any)["properties"].(map[string]any)["steps"].(map[string]any)
	step := steps["additionalProperties"].(map[string]any)
	if required := step["required"].([]string); len(required) != 1 || required[0] != "enabled" {
		t.Fatalf("step schema does not require explicit enabled: %#v", step)
	}
}

func TestMutationAnnotationsMatchTheApprovedToolHints(t *testing.T) {
	for _, annotation := range []struct {
		name        string
		readOnly    bool
		destructive bool
		idempotent  bool
	}{
		{name: ToolReviewVulnerability, readOnly: false, destructive: false, idempotent: true},
		{name: ToolUnreviewVulnerability, readOnly: false, destructive: true, idempotent: true},
	} {
		got := vulnerabilityDispositionAnnotations(annotation.destructive)
		if got.ReadOnlyHint != annotation.readOnly || got.DestructiveHint == nil || *got.DestructiveHint != annotation.destructive || got.IdempotentHint != annotation.idempotent {
			t.Fatalf("%s annotations = %+v", annotation.name, got)
		}
	}
}
