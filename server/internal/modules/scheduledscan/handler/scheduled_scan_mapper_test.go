package handler

import (
	"encoding/json"
	"testing"
	"time"

	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
)

func TestToScheduledScanOutputUsesCanonicalWorkflowAndReturnsTimeZone(t *testing.T) {
	persistedNextRunTime := time.Date(2026, 5, 20, 3, 17, 11, 123000000, time.FixedZone("input", 8*60*60))
	output := toScheduledScanOutput(&scheduledapp.ScheduledScan{
		ID:             1,
		Name:           "daily",
		ScanWorkflowID: "default",
		Configuration:  map[string]any{"steps": map[string]any{}},
		TimeZone:       "UTC", CronExpression: "0 2 * * *",
		IsEnabled:              true,
		SuccessfulHandoffCount: 7,
		FailedHandoffCount:     2,
		NextRunTime:            &persistedNextRunTime,
		CreatedAt:              time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
		UpdatedAt:              time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
	})

	payload, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if decoded["name"] != "scheduledScans/1" || decoded["displayName"] != "daily" {
		t.Fatalf("expected resource name and displayName, got %+v", decoded)
	}
	if decoded["scanWorkflow"] != "scanWorkflows/default" {
		t.Fatalf("expected canonical scanWorkflow, got %+v", decoded)
	}
	if decoded["timeZone"] != "UTC" {
		t.Fatalf("timeZone = %v, want UTC", decoded["timeZone"])
	}
	wantNextRunTime := persistedNextRunTime.UTC().Format(time.RFC3339Nano)
	if decoded["nextRunTime"] != wantNextRunTime {
		t.Fatalf("nextRunTime = %v, want direct persisted UTC projection %q", decoded["nextRunTime"], wantNextRunTime)
	}
	if decoded["successfulHandoffCount"] != float64(7) || decoded["failedHandoffCount"] != float64(2) {
		t.Fatalf("handoff totals = %+v, want 7/2", decoded)
	}
	for _, field := range []string{"workflow", "workflowIds", "workflowId", "workflow_ids"} {
		if _, ok := decoded[field]; ok {
			t.Fatalf("legacy workflow field %q should be omitted: %+v", field, decoded)
		}
	}
}

func TestToScheduledScanOutputProjectsLastHandoffFailureSummary(t *testing.T) {
	failedAt := time.Date(2026, 8, 4, 10, 0, 1, 0, time.UTC)
	cause := scheduledapp.HandoffCauseConfigResourceUnavailable

	// No failure on record: both summary fields stay null together.
	idle := toScheduledScanOutput(&scheduledapp.ScheduledScan{ID: 2})
	if idle.LastHandoffFailureCause != nil || idle.LastHandoffFailureTime != nil {
		t.Fatalf("empty summary = %+v, %+v; want both nil", idle.LastHandoffFailureCause, idle.LastHandoffFailureTime)
	}

	// A settled failure projects the enum cause and the settlement instant.
	failed := toScheduledScanOutput(&scheduledapp.ScheduledScan{
		ID:                      3,
		LastHandoffFailureCause: &cause,
		LastHandoffFailureTime:  &failedAt,
	})
	if failed.LastHandoffFailureCause == nil || *failed.LastHandoffFailureCause != string(cause) {
		t.Fatalf("summary cause = %+v; want %s", failed.LastHandoffFailureCause, cause)
	}
	if failed.LastHandoffFailureTime == nil || !failed.LastHandoffFailureTime.Equal(failedAt) {
		t.Fatalf("summary time = %+v; want %s", failed.LastHandoffFailureTime, failedAt)
	}
}
