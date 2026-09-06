package repository

import (
	"database/sql"
	"strings"
	"testing"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestDecodeEngineExecutionDiagnosticsRejectsUnknownAndTrailingJSON(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "unknown top-level field",
			payload: `{"availability":"unavailable","resultState":"unknown","rawError":"secret"}`,
		},
		{
			name:    "unknown watermark field",
			payload: `{"availability":"available","resultState":"complete","resultTypeWatermarks":[{"resultType":"network.port","receivedItems":1,"encodedItems":1,"submittedItems":1,"acknowledgedItems":1,"submittedBatches":1,"acknowledgedBatches":1,"toolOutput":"secret"}]}`,
		},
		{
			name:    "trailing JSON value",
			payload: `{"availability":"unavailable","resultState":"unknown"} {}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodeEngineExecutionDiagnostics(datatypes.JSON([]byte(test.payload)))
			if err == nil {
				t.Fatal("decodeEngineExecutionDiagnostics() error = nil, want rejection")
			}
		})
	}
}

func TestScanTaskRuntimeRecordRequiresDiagnosticsForEveryTerminalSavedPlan(t *testing.T) {
	row := &scanTaskRuntimeRow{
		ID:                       1,
		Status:                   "failed",
		HasResolvedExecutionPlan: true,
	}
	if _, err := scanTaskRuntimeRowToRecord(row); err == nil || !strings.Contains(err.Error(), "Engine diagnostics are required") {
		t.Fatalf("pre-connect terminal execution = %v, want required diagnostic error", err)
	}

	row = &scanTaskRuntimeRow{
		ID:                       2,
		Status:                   "skipped",
		HasResolvedExecutionPlan: false,
	}
	if _, err := scanTaskRuntimeRowToRecord(row); err != nil {
		t.Fatalf("planning-time skipped task should not require diagnostics: %v", err)
	}
}

func assertPersistedUnavailableEngineDiagnostics(t *testing.T, db *gorm.DB, taskID int) {
	t.Helper()
	var value sql.NullString
	if err := db.Table("scan_task").Select("engine_diagnostics").Where("id = ?", taskID).Scan(&value).Error; err != nil {
		t.Fatalf("read Engine diagnostics for task %d: %v", taskID, err)
	}
	diagnostics, err := decodeEngineExecutionDiagnostics(datatypes.JSON([]byte(value.String)))
	if err != nil {
		t.Fatalf("decode Engine diagnostics for task %d: %v", taskID, err)
	}
	if diagnostics == nil || diagnostics.Availability != scandomain.EngineDiagnosticAvailabilityUnavailable || diagnostics.ResultState != scandomain.EngineDiagnosticResultStateUnknown {
		t.Fatalf("Engine diagnostics for task %d = %#v, want unavailable/unknown", taskID, diagnostics)
	}
}

func assertNoPersistedEngineDiagnostics(t *testing.T, db *gorm.DB, taskID int) {
	t.Helper()
	var value sql.NullString
	if err := db.Table("scan_task").Select("engine_diagnostics").Where("id = ?", taskID).Scan(&value).Error; err != nil {
		t.Fatalf("read Engine diagnostics for task %d: %v", taskID, err)
	}
	if value.Valid {
		t.Fatalf("Engine diagnostics for planning-only task %d = %q, want NULL", taskID, value.String)
	}
	diagnostics, err := decodeEngineExecutionDiagnostics(nil)
	if err != nil {
		t.Fatalf("decode Engine diagnostics for task %d: %v", taskID, err)
	}
	if diagnostics != nil {
		t.Fatalf("Engine diagnostics for planning-only task %d = %#v, want nil", taskID, diagnostics)
	}
}
