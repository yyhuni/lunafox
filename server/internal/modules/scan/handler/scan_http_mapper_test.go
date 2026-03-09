package handler

import (
	"encoding/json"
	"testing"
	"time"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

func TestToScanDetailOutput_UsesConfigurationObjectWithoutYAMLField(t *testing.T) {
	scan := &scanapp.QueryScan{
		ID:          1,
		TargetID:    2,
		WorkflowIDs: []byte(` ["subdomain_discovery"] `),
		Configuration: map[string]any{
			"subdomain_discovery": map[string]any{
				"recon": map[string]any{"enabled": true},
			},
		},
		ScanMode:  "full",
		Status:    "pending",
		CreatedAt: time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC),
	}

	output := toScanDetailOutput(scan)
	payload, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if _, ok := decoded["configuration"]; !ok {
		t.Fatalf("expected configuration field in scan detail output, got %v", decoded)
	}
	if _, ok := decoded["yamlConfiguration"]; ok {
		t.Fatalf("expected yamlConfiguration to be removed from scan detail output, got %v", decoded)
	}
}

func TestToScanOutput_ExposesFailureObject(t *testing.T) {
	scan := &scanapp.QueryScan{
		ID:           11,
		TargetID:     22,
		WorkflowIDs:  []byte(`["subdomain_discovery"]`),
		ScanMode:     "full",
		Status:       "failed",
		ErrorMessage: "task timed out",
		Failure:      &scanapp.FailureDetail{Kind: "task_timeout", Message: "task timed out"},
		CreatedAt:    time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC),
	}

	output := toScanOutput(scan)
	if output.Failure == nil {
		t.Fatalf("expected failure object on scan output")
	}
	if output.Failure.Kind != "task_timeout" || output.Failure.Message != "task timed out" {
		t.Fatalf("unexpected failure output: %+v", output.Failure)
	}
}
