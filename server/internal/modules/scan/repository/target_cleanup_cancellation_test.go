package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCancelNextActiveForDeletedTargetCancelsOneScanAndPreservesTerminalTasks(t *testing.T) {
	db := openTargetCleanupCancellationDB(t)
	repo := NewScanRepository(db)
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	if err := db.Exec("INSERT INTO target (id, deleted_at) VALUES (1, ?), (2, NULL)", now).Error; err != nil {
		t.Fatalf("seed Targets: %v", err)
	}
	if err := db.Exec("INSERT INTO scan (id, target_id, status) VALUES (10, 1, 'pending'), (11, 1, 'running'), (12, 2, 'running')").Error; err != nil {
		t.Fatalf("seed Scans: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO scan_task (id, scan_id, status, resolved_execution_plan) VALUES
			(100, 10, 'blocked', X'01'),
			(101, 10, 'pending', X''),
			(102, 10, 'running', X'01'),
			(103, 10, 'running', X'01'),
			(104, 10, 'succeeded', X'01'),
			(105, 10, 'failed', X'01'),
			(106, 10, 'cancelled', X'01'),
			(201, 11, 'pending', X'01'),
			(301, 12, 'running', X'01')`).Error; err != nil {
		t.Fatalf("seed Tasks: %v", err)
	}
	if err := db.Exec(`UPDATE scan_task
		SET assigned_agent_id = 77, assigned_session_id = 'session-1', assigned_session_epoch = 3, assigned_request_id = 'request-1'
		WHERE id = 102`).Error; err != nil {
		t.Fatalf("seed valid running assignment: %v", err)
	}
	if err := db.Exec("UPDATE scan_task SET assigned_agent_id = 78 WHERE id = 103").Error; err != nil {
		t.Fatalf("seed incomplete running assignment: %v", err)
	}

	first, err := repo.CancelNextActiveForDeletedTarget(context.Background(), 1, now)
	if err != nil || first == nil {
		t.Fatalf("CancelNextActiveForDeletedTarget(first) = %+v, %v", first, err)
	}
	if first.ScanID != 10 || first.CancelledTaskCount != 4 || len(first.NotificationCandidates) != 1 || first.NotificationCandidates[0] != (TargetCleanupTaskCancelCandidate{TaskID: 102, AgentID: 77}) {
		t.Fatalf("first cancellation = %+v", first)
	}
	assertTargetCleanupScanStatus(t, db, 10, "cancelled", true)
	assertTargetCleanupTaskStatuses(t, db, map[int]string{
		100: "cancelled", 101: "cancelled", 102: "cancelled", 103: "cancelled",
		104: "succeeded", 105: "failed", 106: "cancelled",
	})
	assertTargetCleanupTaskCompletion(t, db, []int{100, 101, 102, 103}, true)
	assertTargetCleanupTaskCompletion(t, db, []int{104, 105, 106}, false)
	assertPersistedUnavailableEngineDiagnostics(t, db, 100)
	assertNoPersistedEngineDiagnostics(t, db, 101)
	assertPersistedUnavailableEngineDiagnostics(t, db, 102)
	assertPersistedUnavailableEngineDiagnostics(t, db, 103)

	second, err := repo.CancelNextActiveForDeletedTarget(context.Background(), 1, now.Add(time.Second))
	if err != nil || second == nil || second.ScanID != 11 || second.CancelledTaskCount != 1 {
		t.Fatalf("CancelNextActiveForDeletedTarget(second) = %+v, %v", second, err)
	}
	if third, err := repo.CancelNextActiveForDeletedTarget(context.Background(), 1, now.Add(2*time.Second)); err != nil || third != nil {
		t.Fatalf("CancelNextActiveForDeletedTarget(empty) = %+v, %v", third, err)
	}
	assertTargetCleanupScanStatus(t, db, 12, "running", false)
}

func TestCancelNextActiveForDeletedTargetRequiresTombstone(t *testing.T) {
	db := openTargetCleanupCancellationDB(t)
	repo := NewScanRepository(db)
	if err := db.Exec("INSERT INTO target (id, deleted_at) VALUES (2, NULL)").Error; err != nil {
		t.Fatalf("seed active Target: %v", err)
	}
	if err := db.Exec("INSERT INTO scan (id, target_id, status) VALUES (12, 2, 'running')").Error; err != nil {
		t.Fatalf("seed Scan: %v", err)
	}

	cancellation, err := repo.CancelNextActiveForDeletedTarget(context.Background(), 2, time.Now().UTC())
	if err != nil || cancellation != nil {
		t.Fatalf("active Target cancellation = %+v, %v", cancellation, err)
	}
	assertTargetCleanupScanStatus(t, db, 12, "running", false)
}

func TestCancelNextActiveForDeletedTargetRestartOnlySeesRemainingActiveScans(t *testing.T) {
	db := openTargetCleanupCancellationDB(t)
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	if err := db.Exec("INSERT INTO target (id, deleted_at) VALUES (1, ?)", now).Error; err != nil {
		t.Fatalf("seed tombstoned Target: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO scan (id, target_id, status) VALUES
			(10, 1, 'running'), (11, 1, 'pending'), (12, 1, 'cancelled');
		INSERT INTO scan_task (id, scan_id, status) VALUES
			(100, 10, 'running'), (101, 11, 'pending'), (102, 12, 'succeeded');
	`).Error; err != nil {
		t.Fatalf("seed Scan history: %v", err)
	}

	first, err := NewScanRepository(db).CancelNextActiveForDeletedTarget(context.Background(), 1, now)
	if err != nil || first == nil || first.ScanID != 10 {
		t.Fatalf("first cancellation = %+v, %v", first, err)
	}

	second, err := NewScanRepository(db).CancelNextActiveForDeletedTarget(context.Background(), 1, now.Add(time.Second))
	if err != nil || second == nil || second.ScanID != 11 {
		t.Fatalf("restarted cancellation = %+v, %v", second, err)
	}
	third, err := NewScanRepository(db).CancelNextActiveForDeletedTarget(context.Background(), 1, now.Add(2*time.Second))
	if err != nil || third != nil {
		t.Fatalf("remaining cancellation = %+v, %v", third, err)
	}
	assertTargetCleanupScanStatus(t, db, 10, "cancelled", true)
	assertTargetCleanupScanStatus(t, db, 11, "cancelled", true)
	assertTargetCleanupScanStatus(t, db, 12, "cancelled", false)
	assertTargetCleanupTaskStatuses(t, db, map[int]string{100: "cancelled", 101: "cancelled", 102: "succeeded"})
	assertTargetCleanupTaskCompletion(t, db, []int{100, 101}, true)
	assertTargetCleanupTaskCompletion(t, db, []int{102}, false)
}

func openTargetCleanupCancellationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	for _, statement := range []string{
		"CREATE TABLE target (id INTEGER PRIMARY KEY, deleted_at DATETIME)",
		"CREATE TABLE scan (id INTEGER PRIMARY KEY, target_id INTEGER NOT NULL, status TEXT NOT NULL, stopped_at DATETIME)",
		`CREATE TABLE scan_task (
			id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL, status TEXT NOT NULL,
			assigned_agent_id INTEGER, assigned_session_id TEXT, assigned_session_epoch INTEGER, assigned_request_id TEXT,
			completed_at DATETIME,
			resolved_execution_plan BLOB NOT NULL DEFAULT '',
			engine_diagnostics TEXT
		)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create fixture table: %v", err)
		}
	}
	return db
}

func assertTargetCleanupScanStatus(t *testing.T, db *gorm.DB, scanID int, want string, wantStopped bool) {
	t.Helper()
	var row struct {
		Status    string
		StoppedAt *time.Time `gorm:"column:stopped_at"`
	}
	if err := db.Table("scan").Where("id = ?", scanID).Take(&row).Error; err != nil {
		t.Fatalf("read Scan %d: %v", scanID, err)
	}
	if row.Status != want || (row.StoppedAt != nil) != wantStopped {
		t.Fatalf("Scan %d = %+v, want status=%q stopped=%v", scanID, row, want, wantStopped)
	}
}

func assertTargetCleanupTaskStatuses(t *testing.T, db *gorm.DB, want map[int]string) {
	t.Helper()
	for id, status := range want {
		var actual string
		if err := db.Table("scan_task").Select("status").Where("id = ?", id).Scan(&actual).Error; err != nil {
			t.Fatalf("read Task %d: %v", id, err)
		}
		if actual != status {
			t.Fatalf("Task %d status = %q, want %q", id, actual, status)
		}
	}
}

func assertTargetCleanupTaskCompletion(t *testing.T, db *gorm.DB, taskIDs []int, want bool) {
	t.Helper()
	for _, id := range taskIDs {
		var count int64
		if err := db.Table("scan_task").Where("id = ? AND completed_at IS NOT NULL", id).Count(&count).Error; err != nil {
			t.Fatalf("read Task %d completion: %v", id, err)
		}
		if (count == 1) != want {
			t.Fatalf("Task %d completed_at set=%v, want set=%v", id, count == 1, want)
		}
	}
}
