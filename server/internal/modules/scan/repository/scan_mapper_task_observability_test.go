package repository

import (
	"testing"

	"gorm.io/datatypes"

	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
)

func TestScanTaskModelToRecord_MapsCoreFields(t *testing.T) {
	item := &model.ScanTask{
		ID:             91,
		ScanID:         19,
		Stage:          2,
		WorkflowID:     "subdomain_discovery",
		WorkflowConfig: datatypes.JSON([]byte(`{"recon":{"enabled":false}}`)),
		FailureKind:    "decode_config_failed",
	}

	record := scanTaskModelToRecord(item)
	if record == nil {
		t.Fatalf("expected record")
	}
	if record.ID != 91 || record.ScanID != 19 || record.Stage != 2 {
		t.Fatalf("unexpected mapped core fields: %+v", record)
	}
	if record.WorkflowID != "subdomain_discovery" {
		t.Fatalf("unexpected workflow id: %s", record.WorkflowID)
	}
	if record.WorkflowConfig == nil {
		t.Fatalf("expected workflow config object to be mapped")
	}
	if record.FailureKind != "decode_config_failed" {
		t.Fatalf("unexpected failure kind: %s", record.FailureKind)
	}
}
