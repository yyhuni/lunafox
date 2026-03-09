package repository

import (
	"testing"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestScanRepositoryUpdateStatus_NormalizesEmptyFailureKindToUnknown(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_update_status_failure_kind?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE scan (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			status TEXT NOT NULL,
			error_message TEXT NOT NULL DEFAULT '',
			failure_kind TEXT NOT NULL DEFAULT '',
			stopped_at DATETIME
		);
		INSERT INTO scan (id, status, error_message, failure_kind) VALUES (1, 'running', '', '');
	`).Error; err != nil {
		t.Fatalf("setup scan table failed: %v", err)
	}

	repo := &ScanRepository{db: db}
	failure := &scandomain.FailureDetail{Kind: "", Message: "task timed out"}
	if err := repo.UpdateStatus(1, scanStatusFailed, failure); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	var row struct {
		Status       string
		ErrorMessage string
		FailureKind  string
	}
	if err := db.Table("scan").Select("status, error_message, failure_kind").Where("id = 1").Scan(&row).Error; err != nil {
		t.Fatalf("query scan failed: %v", err)
	}
	if row.Status != scanStatusFailed || row.ErrorMessage != "task timed out" || row.FailureKind != "unknown" {
		t.Fatalf("unexpected scan row: %+v", row)
	}
}
