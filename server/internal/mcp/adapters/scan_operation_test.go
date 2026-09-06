package adapters

import (
	"testing"
	"time"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

func TestOperationRecordProjectsTerminalResponseOrErrorExclusively(t *testing.T) {
	now := time.Date(2026, 8, 1, 2, 0, 0, 0, time.UTC)
	base := scanapp.MCPScanOperation{
		ID: "11111111-1111-1111-1111-111111111111", ScanID: 4, TargetID: 2,
		Progress: 44, CreatedAt: now, UpdatedAt: now,
	}
	for _, test := range []struct {
		status       string
		progress     int
		wantResponse bool
		wantError    string
	}{
		{status: "pending", progress: 0},
		{status: "running", progress: 44},
		{status: "succeeded", progress: 100, wantResponse: true},
		{status: "failed", progress: 44, wantError: "SCAN_FAILED"},
		{status: "cancelled", progress: 44, wantError: "SCAN_CANCELLED"},
	} {
		t.Run(test.status, func(t *testing.T) {
			operation := base
			operation.Status = test.status
			operation.Progress = test.progress
			record := operationRecord(operation)
			if record.Name != "operations/"+operation.ID || record.Scan != "scans/4" || record.Target != "targets/2" || record.Status != stringUpper(test.status) || record.Progress != test.progress {
				t.Fatalf("operation record = %+v", record)
			}
			if (record.Response != nil) != test.wantResponse || (record.Error != nil) != (test.wantError != "") {
				t.Fatalf("terminal branch = %+v", record)
			}
			if test.wantError != "" && record.Error["code"] != test.wantError {
				t.Fatalf("terminal error = %+v", record.Error)
			}
		})
	}
}

func stringUpper(value string) string {
	if value == "pending" {
		return "PENDING"
	}
	if value == "running" {
		return "RUNNING"
	}
	if value == "succeeded" {
		return "SUCCEEDED"
	}
	if value == "failed" {
		return "FAILED"
	}
	return "CANCELLED"
}
