package repository

import (
	"reflect"
	"testing"

	"gorm.io/datatypes"
)

func TestScheduledScanReadPreservesPersistedConfigurationWithoutEnablementInference(t *testing.T) {
	persisted := map[string]any{
		"steps": map[string]any{
			"discover": map[string]any{"enabled": false},
			"ports": map[string]any{
				"enabled":      true,
				"engineConfig": map[string]any{"scan": map[string]any{"enabled": true, "timeout": 60}},
			},
		},
	}
	persisted["steps"].(map[string]any)["ports"].(map[string]any)["engineConfig"].(map[string]any)["scan"].(map[string]any)["timeout"] = float64(60)
	record, err := scheduledScanModelToRecord(&scheduledScanModel{
		ID:                     11,
		ScanWorkflowID:         "default",
		Configuration:          datatypes.JSON([]byte(`{"steps":{"discover":{"enabled":false},"ports":{"enabled":true,"engineConfig":{"scan":{"enabled":true,"timeout":60}}}}}`)),
		InputSource:            "scan_snapshot",
		IsEnabled:              true,
		SuccessfulHandoffCount: 3,
		FailedHandoffCount:     2,
	})
	if err != nil {
		t.Fatalf("scheduledScanModelToRecord() error = %v", err)
	}
	if record == nil || !reflect.DeepEqual(record.Configuration, persisted) {
		t.Fatalf("scheduled scan read changed persisted configuration: got=%#v want=%#v", record, persisted)
	}
	if record.SuccessfulHandoffCount != 3 || record.FailedHandoffCount != 2 {
		t.Fatalf("scheduled scan read changed persisted handoff totals: %+v", record)
	}
	if record.IsEnabled {
		// Management enablement is returned as its own persisted field; it must
		// not be used to rewrite the stored Workflow Step branches.
		return
	}
	t.Fatalf("scheduled scan isEnabled value was not preserved: %+v", record)
}
