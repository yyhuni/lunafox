package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCreateScheduledScanRequestRejectsLegacyWorkflowFields(t *testing.T) {
	cases := map[string]string{
		"workflowIds":      `{"displayName":"daily","workflowIds":["subdomain_discovery"],"configuration":{},"cronExpression":"0 2 * * *"}`,
		"workflowId":       `{"displayName":"daily","workflowId":"subdomain_discovery","configuration":{},"cronExpression":"0 2 * * *"}`,
		"workflow":         `{"displayName":"daily","workflow":"scanWorkflows/default","configuration":{},"cronExpression":"0 2 * * *"}`,
		"defaultProfileId": `{"displayName":"daily","scanWorkflow":"scanWorkflows/default","defaultProfileId":"legacy","configuration":{},"cronExpression":"0 2 * * *"}`,
		"configSchemaId":   `{"displayName":"daily","scanWorkflow":"scanWorkflows/default","configSchemaId":"schema.workflow.default","configuration":{},"cronExpression":"0 2 * * *"}`,
	}
	for field, payload := range cases {
		t.Run(field, func(t *testing.T) {
			var request CreateScheduledScanRequest
			err := json.Unmarshal([]byte(payload), &request)
			if err == nil || !strings.Contains(err.Error(), field) {
				t.Fatalf("expected %s rejection, got %v", field, err)
			}
		})
	}
}

func TestCreateScheduledScanRequestAcceptsCanonicalScanWorkflowConfiguration(t *testing.T) {
	payload := []byte(`{"displayName":"daily","scanWorkflow":"scanWorkflows/default","configuration":{"steps":{"subdomain_discovery":{"engineConfig":{}}}},"target":"targets/7","cronExpression":"0 2 * * *","isEnabled":true}`)
	var request CreateScheduledScanRequest

	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatalf("expected canonical request to decode, got %v", err)
	}
	if request.DisplayName != "daily" {
		t.Fatalf("unexpected displayName: %+v", request)
	}
	if request.ScanWorkflow != "scanWorkflows/default" {
		t.Fatalf("unexpected scan workflow: %+v", request)
	}
	if _, ok := request.Configuration["steps"]; !ok {
		t.Fatalf("expected object configuration, got %+v", request.Configuration)
	}
}

func TestCreateScheduledScanRequestRejectsRetiredTimeZone(t *testing.T) {
	var request CreateScheduledScanRequest
	err := json.Unmarshal([]byte(`{"displayName":"daily","scanWorkflow":"scanWorkflows/default","configuration":{},"cronExpression":"0 2 * * *","timeZone":"UTC"}`), &request)
	if err == nil || !strings.Contains(err.Error(), "timeZone") {
		t.Fatalf("expected retired timeZone rejection, got %v", err)
	}
}

func TestBatchUpdateScheduledScansRequestRequiresOnlyCanonicalFields(t *testing.T) {
	payload := []byte(`{"requests":[{"name":"scheduledScans/1","isEnabled":false,"updateMask":"isEnabled"}]}`)
	var request BatchUpdateScheduledScansRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatalf("expected batch request to decode, got %v", err)
	}
	if len(request.Requests) != 1 || request.Requests[0].IsEnabled == nil || *request.Requests[0].IsEnabled {
		t.Fatalf("unexpected batch request: %+v", request)
	}

	for _, payload := range []string{
		`{"requests":[{"name":"scheduledScans/1","isEnabled":true,"updateMask":"isEnabled","displayName":"invalid"}]}`,
		`{"requests":[],"unexpected":true}`,
	} {
		var invalid BatchUpdateScheduledScansRequest
		if err := json.Unmarshal([]byte(payload), &invalid); err == nil {
			t.Fatalf("expected strict batch field validation for %s", payload)
		}
	}
}

func TestBatchUpdateScheduledScansResponseUsesUpdatedCount(t *testing.T) {
	payload, err := json.Marshal(BatchUpdateScheduledScansResponse{UpdatedCount: 2})
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(decoded) != 1 || decoded["updatedCount"] != float64(2) {
		t.Fatalf("unexpected batch response: %+v", decoded)
	}
}

func TestScheduledScanResponseSerializesOnlyCanonicalScanWorkflow(t *testing.T) {
	response := ScheduledScanResponse{
		ID:                     1,
		Name:                   "scheduledScans/1",
		DisplayName:            "daily",
		ScanWorkflow:           "scanWorkflows/default",
		Configuration:          map[string]any{"steps": map[string]any{}},
		CronExpression:         "0 2 * * *",
		IsEnabled:              true,
		SuccessfulHandoffCount: 3,
		FailedHandoffCount:     1,
		CreatedAt:              time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
		UpdatedAt:              time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
	}

	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if decoded["scanWorkflow"] != "scanWorkflows/default" {
		t.Fatalf("expected canonical scanWorkflow field, got %+v", decoded)
	}
	if _, ok := decoded["timeZone"]; ok {
		t.Fatalf("retired timeZone field must be omitted: %+v", decoded)
	}
	if nextRunTime, ok := decoded["nextRunTime"]; !ok || nextRunTime != nil {
		t.Fatalf("expected explicit null nextRunTime, got %+v", decoded)
	}
	if lastRunTime, ok := decoded["lastRunTime"]; !ok || lastRunTime != nil {
		t.Fatalf("expected explicit null lastRunTime, got %+v", decoded)
	}
	if decoded["name"] != "scheduledScans/1" || decoded["displayName"] != "daily" {
		t.Fatalf("expected canonical name and displayName, got %+v", decoded)
	}
	if decoded["successfulHandoffCount"] != float64(3) || decoded["failedHandoffCount"] != float64(1) {
		t.Fatalf("expected required handoff totals, got %+v", decoded)
	}
	for _, field := range []string{"workflow", "workflowIds", "workflowId", "workflow_ids"} {
		if _, ok := decoded[field]; ok {
			t.Fatalf("legacy workflow field %q should be omitted: %+v", field, decoded)
		}
	}
}

func TestScheduledScanListResponseUsesResourceSpecificCollection(t *testing.T) {
	response := NewScheduledScanListResponse([]ScheduledScanResponse{
		{Name: "scheduledScans/1", DisplayName: "daily"},
	}, 1, 1, 20)

	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := decoded["scheduledScans"]; !ok {
		t.Fatalf("expected scheduledScans collection, got %+v", decoded)
	}
	if _, ok := decoded["results"]; ok {
		t.Fatalf("generic results collection should not be used: %+v", decoded)
	}
	if decoded["totalSize"] != float64(1) {
		t.Fatalf("expected totalSize, got %+v", decoded)
	}
}
