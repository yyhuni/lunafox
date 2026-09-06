package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newBatchStopRepositoryForTest(t *testing.T) *ScanRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:scan_batch_stop_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE scan (
			id INTEGER PRIMARY KEY,
			status TEXT NOT NULL,
			deleted_at DATETIME,
			stopped_at DATETIME
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
			completed_at DATETIME
		);
	`).Error; err != nil {
		t.Fatalf("create batch stop schema: %v", err)
	}
	return &ScanRepository{db: db}
}

func seedBatchStopRows(t *testing.T, repo *ScanRepository) {
	t.Helper()
	if err := repo.db.Exec(`
		INSERT INTO scan (id, status) VALUES
			(1, 'running'),
			(2, 'pending'),
			(3, 'succeeded');
		INSERT INTO scan_task (id, scan_id, status, assigned_agent_id, assigned_session_id, assigned_session_epoch, assigned_request_id, resolved_execution_plan)
		VALUES
			(11, 1, 'running', 7, 'session-1', 3, 'request-1', X'01'),
			(12, 1, 'pending', NULL, NULL, NULL, NULL, X'01'),
			(21, 2, 'blocked', NULL, NULL, NULL, NULL, X'01'),
			(31, 3, 'succeeded', NULL, NULL, NULL, NULL, X'01');
	`).Error; err != nil {
		t.Fatalf("seed batch stop rows: %v", err)
	}
}

func TestScanRepositoryBatchStopActiveScansHandlesMixedStatesAndNotifications(t *testing.T) {
	repo := newBatchStopRepositoryForTest(t)
	seedBatchStopRows(t, repo)
	stoppedAt := time.Date(2026, 8, 18, 12, 30, 0, 0, time.UTC)

	outcome, err := repo.BatchStopActiveScans(context.Background(), []int{3, 1, 2}, stoppedAt)
	if err != nil {
		t.Fatalf("BatchStopActiveScans: %v", err)
	}
	if outcome.StoppedCount != 2 || outcome.SkippedCount != 1 || outcome.RevokedTaskCount != 3 {
		t.Fatalf("unexpected batch outcome: %+v", outcome)
	}
	if len(outcome.NotificationCandidates) != 1 || outcome.NotificationCandidates[0].ScanID != 1 || outcome.NotificationCandidates[0].TaskID != 11 {
		t.Fatalf("unexpected notification candidates: %+v", outcome.NotificationCandidates)
	}

	var scans []struct {
		ID        int
		Status    string
		StoppedAt *time.Time
	}
	if err := repo.db.Table("scan").Select("id, status, stopped_at").Order("id ASC").Find(&scans).Error; err != nil {
		t.Fatalf("read scans: %v", err)
	}
	if scans[0].Status != scanStatusCancelled || scans[1].Status != scanStatusCancelled || scans[2].Status != scanStatusSucceeded {
		t.Fatalf("unexpected scan statuses: %+v", scans)
	}
	if scans[0].StoppedAt == nil || !scans[0].StoppedAt.Equal(stoppedAt) || scans[1].StoppedAt == nil || !scans[1].StoppedAt.Equal(stoppedAt) {
		t.Fatalf("batch stop time mismatch: %+v", scans)
	}

	var cancelledTasks int64
	if err := repo.db.Table("scan_task").Where("scan_id IN ? AND status = ?", []int{1, 2}, taskStatusCancelled).Count(&cancelledTasks).Error; err != nil {
		t.Fatalf("count cancelled tasks: %v", err)
	}
	if cancelledTasks != 3 {
		t.Fatalf("cancelled task count = %d, want 3", cancelledTasks)
	}
}

func TestScanRepositoryBatchStopActiveScansIsIdempotentForTerminalReplay(t *testing.T) {
	repo := newBatchStopRepositoryForTest(t)
	seedBatchStopRows(t, repo)
	stoppedAt := time.Date(2026, 8, 18, 12, 30, 0, 0, time.UTC)
	if _, err := repo.BatchStopActiveScans(context.Background(), []int{1, 2}, stoppedAt); err != nil {
		t.Fatalf("first BatchStopActiveScans: %v", err)
	}

	replay, err := repo.BatchStopActiveScans(context.Background(), []int{2, 1}, stoppedAt.Add(time.Minute))
	if err != nil {
		t.Fatalf("replayed BatchStopActiveScans: %v", err)
	}
	if replay.StoppedCount != 0 || replay.SkippedCount != 2 || replay.RevokedTaskCount != 0 {
		t.Fatalf("unexpected replay outcome: %+v", replay)
	}
	var scan struct {
		StoppedAt *time.Time
	}
	if err := repo.db.Table("scan").Select("stopped_at").Where("id = ?", 1).Scan(&scan).Error; err != nil {
		t.Fatalf("read replayed scan: %v", err)
	}
	if scan.StoppedAt == nil || !scan.StoppedAt.Equal(stoppedAt) {
		t.Fatalf("replay changed stop time: %v", scan.StoppedAt)
	}
}

func TestScanRepositoryBatchStopActiveScansRollsBackWhenOneScanIsMissing(t *testing.T) {
	repo := newBatchStopRepositoryForTest(t)
	seedBatchStopRows(t, repo)

	_, err := repo.BatchStopActiveScans(context.Background(), []int{1, 999}, time.Now().UTC())
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing scan error = %v, want record not found", err)
	}
	var status string
	if err := repo.db.Table("scan").Select("status").Where("id = ?", 1).Scan(&status).Error; err != nil {
		t.Fatalf("read rolled-back scan: %v", err)
	}
	if status != scanStatusRunning {
		t.Fatalf("missing-scan batch changed scan status to %q", status)
	}
}

func TestScanRepositoryBatchStopActiveScansRejectsDuplicateIDs(t *testing.T) {
	repo := newBatchStopRepositoryForTest(t)
	_, err := repo.BatchStopActiveScans(context.Background(), []int{1, 1}, time.Now().UTC())
	if err == nil {
		t.Fatal("duplicate batch IDs unexpectedly succeeded")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("duplicate IDs returned record-not-found: %v", err)
	}
}
