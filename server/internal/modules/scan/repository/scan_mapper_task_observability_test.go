package repository

import (
	"testing"

	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
)

func TestScanTaskModelToRecord_MapsCoreFields(t *testing.T) {
	item := &model.ScanTask{
		ID:           91,
		ScanID:       19,
		Stage:        2,
		WorkflowName: "subdomain_discovery",
	}

	record := scanTaskModelToRecord(item)
	if record == nil {
		t.Fatalf("expected record")
	}
	if record.ID != 91 || record.ScanID != 19 || record.Stage != 2 {
		t.Fatalf("unexpected mapped core fields: %+v", record)
	}
	if record.WorkflowName != "subdomain_discovery" {
		t.Fatalf("unexpected workflow name: %s", record.WorkflowName)
	}
}
