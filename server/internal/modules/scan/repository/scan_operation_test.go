package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetMCPOperationProjectsScanLifecycleWithoutSecondStateMachine(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan-mcp-operation-projection?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	statements := []string{
		`CREATE TABLE scan (
			id INTEGER PRIMARY KEY, status TEXT NOT NULL, progress INTEGER NOT NULL,
			current_stage TEXT NOT NULL, failure_kind TEXT NOT NULL,
			created_at DATETIME NOT NULL, stopped_at DATETIME
		)`,
		`CREATE TABLE scan_operation (
			id TEXT PRIMARY KEY, scan_id INTEGER NOT NULL, target_id INTEGER NOT NULL,
			request_fingerprint TEXT NOT NULL, created_at DATETIME NOT NULL
		)`,
		`CREATE TABLE scan_task (
			id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL, status TEXT NOT NULL,
			stage_order INTEGER NOT NULL, step_order INTEGER NOT NULL, step_id TEXT NOT NULL,
			created_at DATETIME NOT NULL, started_at DATETIME, completed_at DATETIME
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare operation table: %v", err)
		}
	}
	createdAt := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	startedAt := createdAt.Add(2 * time.Minute)
	if err := db.Exec(`INSERT INTO scan (id, status, progress, current_stage, failure_kind, created_at) VALUES (1, 'running', 37, 'discovery', '', ?)`, createdAt).Error; err != nil {
		t.Fatalf("insert scan: %v", err)
	}
	if err := db.Exec(`INSERT INTO scan_operation (id, scan_id, target_id, request_fingerprint, created_at) VALUES ('operation-1', 1, 9, 'fingerprint', ?)`, createdAt.Add(time.Minute)).Error; err != nil {
		t.Fatalf("insert operation: %v", err)
	}
	if err := db.Exec(`INSERT INTO scan_task (id, scan_id, status, stage_order, step_order, step_id, created_at) VALUES (10, 1, 'pending', 2, 0, 'later', ?)`, createdAt).Error; err != nil {
		t.Fatalf("insert pending task: %v", err)
	}
	if err := db.Exec(`INSERT INTO scan_task (id, scan_id, status, stage_order, step_order, step_id, created_at, started_at) VALUES (11, 1, 'running', 1, 0, 'active', ?, ?)`, createdAt, startedAt).Error; err != nil {
		t.Fatalf("insert running task: %v", err)
	}

	record, err := NewScanRepository(db).GetMCPOperation(context.Background(), "operation-1")
	if err != nil {
		t.Fatalf("GetMCPOperation running: %v", err)
	}
	if record.ScanStatus != "running" || record.Progress != 37 || record.Phase != "discovery" || record.CurrentTask != "active" || !record.UpdatedAt.Equal(startedAt) {
		t.Fatalf("running operation projection = %+v", record)
	}

	stoppedAt := startedAt.Add(3 * time.Minute)
	if err := db.Exec(`UPDATE scan SET status = 'succeeded', progress = 4, stopped_at = ? WHERE id = 1`, stoppedAt).Error; err != nil {
		t.Fatalf("complete scan: %v", err)
	}
	if err := db.Exec(`UPDATE scan_task SET status = 'succeeded', completed_at = ? WHERE id = 11`, stoppedAt).Error; err != nil {
		t.Fatalf("complete task: %v", err)
	}
	if err := db.Exec(`UPDATE scan_task SET status = 'succeeded', completed_at = ? WHERE id = 10`, stoppedAt).Error; err != nil {
		t.Fatalf("complete pending task: %v", err)
	}
	record, err = NewScanRepository(db).GetMCPOperation(context.Background(), "operation-1")
	if err != nil {
		t.Fatalf("GetMCPOperation succeeded: %v", err)
	}
	if record.ScanStatus != "succeeded" || record.Progress != 100 || record.CurrentTask != "" || !record.UpdatedAt.Equal(stoppedAt) {
		t.Fatalf("succeeded operation projection = %+v", record)
	}
	if _, err := NewScanRepository(db).GetMCPOperation(context.Background(), "missing"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing operation error = %v", err)
	}
}

func TestMCPOperationProjectionClampsProgressAndKeepsLatestTimestamp(t *testing.T) {
	if got := clampMCPOperationProgress("running", -1); got != 0 {
		t.Fatalf("negative progress = %d", got)
	}
	if got := clampMCPOperationProgress("running", 101); got != 100 {
		t.Fatalf("overflow progress = %d", got)
	}
	if got := clampMCPOperationProgress("succeeded", 0); got != 100 {
		t.Fatalf("succeeded progress = %d", got)
	}
	first := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)
	latest := first.Add(time.Minute)
	if got := latestMCPOperationTime(first, first.Add(-time.Minute), latest); !got.Equal(latest) {
		t.Fatalf("latest operation timestamp = %s, want %s", got, latest)
	}
}
