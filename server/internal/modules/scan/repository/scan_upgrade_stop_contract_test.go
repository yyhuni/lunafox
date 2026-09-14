package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStopAllActiveScansForUpgradeCancelsActiveRowsWithAuditFence(t *testing.T) {
	dsn := fmt.Sprintf("file:scan_upgrade_stop_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, closeErr := db.DB(); closeErr == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.Exec(`
		CREATE TABLE scan (
			id INTEGER PRIMARY KEY,
			status TEXT NOT NULL,
			deleted_at DATETIME,
			stopped_at DATETIME,
			cancellation_reason TEXT,
			cancellation_operation_id TEXT
		);
		CREATE TABLE scan_task (
			id INTEGER PRIMARY KEY,
			scan_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			assigned_agent_id INTEGER,
			assigned_session_id TEXT,
			assigned_session_epoch INTEGER,
			assigned_request_id TEXT,
			resolved_execution_plan BLOB,
			engine_diagnostics TEXT,
			completed_at DATETIME,
			cancellation_reason TEXT,
			cancellation_operation_id TEXT
		);
	`).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO scan (id, status) VALUES
			(1, 'running'), (2, 'pending'), (3, 'succeeded'), (4, 'failed');
		INSERT INTO scan_task (id, scan_id, status, assigned_agent_id, assigned_session_id, assigned_session_epoch, assigned_request_id, resolved_execution_plan)
		VALUES
			(11, 1, 'running', 7, 'session-7', 3, 'request-11', X'01'),
			(12, 1, 'pending', NULL, NULL, NULL, NULL, X'01'),
			(21, 2, 'blocked', NULL, NULL, NULL, NULL, X'01'),
			(22, 2, 'running', 8, 'session-8', 4, 'request-22', X'01'),
			(31, 3, 'succeeded', 9, 'session-9', 5, 'request-31', X'01'),
			(41, 4, 'failed', NULL, NULL, NULL, NULL, X'01');
	`).Error; err != nil {
		t.Fatalf("seed rows: %v", err)
	}

	repository := &ScanRepository{db: db}
	operationID := "11111111-1111-4111-8111-111111111111"
	stoppedAt := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	outcome, err := repository.StopAllActiveScansForUpgrade(context.Background(), operationID, stoppedAt)
	if err != nil {
		t.Fatalf("StopAllActiveScansForUpgrade() error = %v", err)
	}
	if outcome.CancelledScanCount != 2 || outcome.CancelledTaskCount != 4 {
		t.Fatalf("outcome = %+v, want 2 scans and 4 tasks", outcome)
	}
	if len(outcome.NotificationCandidates) != 2 || outcome.NotificationCandidates[0].TaskID != 11 || outcome.NotificationCandidates[1].TaskID != 22 {
		t.Fatalf("notification candidates = %+v", outcome.NotificationCandidates)
	}

	var scans []struct {
		ID          int
		Status      string
		Reason      string     `gorm:"column:cancellation_reason"`
		OperationID string     `gorm:"column:cancellation_operation_id"`
		StoppedAt   *time.Time `gorm:"column:stopped_at"`
	}
	if err := db.Table("scan").Select("id, status, cancellation_reason, cancellation_operation_id, stopped_at").Order("id").Find(&scans).Error; err != nil {
		t.Fatalf("read scans: %v", err)
	}
	for _, scan := range scans {
		if scan.ID <= 2 {
			if scan.Status != scanStatusCancelled || scan.Reason != UpgradeCancellationReason || scan.OperationID != operationID || scan.StoppedAt == nil || !scan.StoppedAt.Equal(stoppedAt) {
				t.Fatalf("active scan audit = %+v", scan)
			}
			continue
		}
		if scan.Status == scanStatusCancelled {
			t.Fatalf("terminal scan %d was cancelled: %+v", scan.ID, scan)
		}
	}

	var tasks []struct {
		ID          int
		Status      string
		Reason      string     `gorm:"column:cancellation_reason"`
		OperationID string     `gorm:"column:cancellation_operation_id"`
		CompletedAt *time.Time `gorm:"column:completed_at"`
		Diagnostics string     `gorm:"column:engine_diagnostics"`
	}
	if err := db.Table("scan_task").Select("id, status, cancellation_reason, cancellation_operation_id, completed_at, engine_diagnostics").Order("id").Find(&tasks).Error; err != nil {
		t.Fatalf("read tasks: %v", err)
	}
	for _, task := range tasks {
		if task.ID <= 22 {
			if task.Status != taskStatusCancelled || task.Reason != UpgradeCancellationReason || task.OperationID != operationID || task.CompletedAt == nil {
				t.Fatalf("active task audit = %+v", task)
			}
			if !strings.Contains(task.Diagnostics, "unavailable") {
				t.Fatalf("active task diagnostics = %q", task.Diagnostics)
			}
		} else if task.Status == taskStatusCancelled {
			t.Fatalf("terminal task %d was cancelled", task.ID)
		}
	}

	// A second maintenance pass has no active rows to mutate and cannot create
	// duplicate notification candidates.
	replay, err := repository.StopAllActiveScansForUpgrade(context.Background(), operationID, stoppedAt.Add(time.Minute))
	if err != nil {
		t.Fatalf("replay error = %v", err)
	}
	if replay.CancelledScanCount != 0 || replay.CancelledTaskCount != 0 || len(replay.NotificationCandidates) != 0 {
		t.Fatalf("replay outcome = %+v, want empty", replay)
	}
}
