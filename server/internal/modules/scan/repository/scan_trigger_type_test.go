package repository

import (
	"context"
	"errors"
	"testing"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestScanTriggerTypeMappingsFailClosed(t *testing.T) {
	for _, triggerType := range []string{"", "legacy", "unknown"} {
		t.Run(triggerType, func(t *testing.T) {
			if _, err := scanModelToRecord(&model.Scan{TriggerType: triggerType}); !errors.Is(err, scandomain.ErrInvalidScanTriggerType) {
				t.Fatalf("scanModelToRecord(%q) error = %v, want invalid trigger type", triggerType, err)
			}
			if _, err := scanCreateRecordToModel(&ScanCreateRecord{TriggerType: scandomain.ScanTriggerType(triggerType)}); !errors.Is(err, scandomain.ErrInvalidScanTriggerType) {
				t.Fatalf("scanCreateRecordToModel(%q) error = %v, want invalid trigger type", triggerType, err)
			}
		})
	}
}

func TestScanRepositoryScheduledTriggerSurvivesScheduleAndOccurrenceDeletion(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan_trigger_schedule_deletion?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE scheduled_scan (id INTEGER PRIMARY KEY);
		CREATE TABLE scheduled_scan_occurrence (id INTEGER PRIMARY KEY, scheduled_scan_id INTEGER NOT NULL);
		CREATE TABLE scan (
			id INTEGER PRIMARY KEY,
			input_source TEXT NOT NULL CHECK (input_source IN ('scan_snapshot', 'target_inventory')),
			trigger_type TEXT NOT NULL CHECK (trigger_type IN ('manual', 'scheduled', 'ai')),
			deleted_at DATETIME
		);
		INSERT INTO scheduled_scan (id) VALUES (7);
		INSERT INTO scheduled_scan_occurrence (id, scheduled_scan_id) VALUES (13, 7);
		INSERT INTO scan (id, input_source, trigger_type) VALUES (91, 'scan_snapshot', 'scheduled');
	`).Error; err != nil {
		t.Fatalf("seed provenance fixtures: %v", err)
	}

	if err := db.WithContext(context.Background()).Exec("DELETE FROM scheduled_scan_occurrence WHERE id = ?", 13).Error; err != nil {
		t.Fatalf("delete occurrence: %v", err)
	}
	if err := db.WithContext(context.Background()).Exec("DELETE FROM scheduled_scan WHERE id = ?", 7).Error; err != nil {
		t.Fatalf("delete schedule: %v", err)
	}

	scan, err := NewScanRepository(db).GetByID(91)
	if err != nil {
		t.Fatalf("read Scan after Schedule deletion: %v", err)
	}
	if scan.TriggerType != scandomain.ScanTriggerTypeScheduled {
		t.Fatalf("Scan trigger type after Schedule deletion = %q, want scheduled", scan.TriggerType)
	}
}
