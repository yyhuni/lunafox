package repository

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTaskProgressLogRepositoryBatchCreateTaskProgressLogs_UsesRequestIDForIdempotency(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:task_progress_log_request_id?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE scan_task (
			id INTEGER PRIMARY KEY,
			scan_id INTEGER NOT NULL,
			UNIQUE (scan_id, id)
		);
		CREATE TABLE task_progress_log (
			id INTEGER NOT NULL,
			scan_id INTEGER NOT NULL,
			task_id INTEGER NOT NULL,
			request_id TEXT NOT NULL DEFAULT '',
			sequence INTEGER NOT NULL DEFAULT 0,
			level TEXT NOT NULL DEFAULT 'info',
			content TEXT NOT NULL DEFAULT '',
			emitted_at DATETIME,
			created_at DATETIME,
			PRIMARY KEY (scan_id, id),
			FOREIGN KEY (scan_id, task_id) REFERENCES scan_task(scan_id, id) ON DELETE CASCADE
		);
		CREATE UNIQUE INDEX uniq_task_progress_log_request ON task_progress_log(scan_id, task_id, request_id, sequence);
	`).Error; err != nil {
		t.Fatalf("setup task_progress_log table failed: %v", err)
	}

	taskID := 101
	emittedAt := time.Date(2026, 5, 7, 10, 30, 0, 0, time.UTC)
	if err := db.Exec("INSERT INTO scan_task (id, scan_id) VALUES (?, ?)", taskID, 7).Error; err != nil {
		t.Fatalf("insert scan task failed: %v", err)
	}
	repo := NewTaskProgressLogRepository(db)
	accepted, duplicates, err := repo.BatchCreateTaskProgressLogs(context.Background(), []TaskProgressLogRecord{{
		ID: 1, ScanID: 7, TaskID: taskID, RequestID: "request-1", Sequence: 1, Level: "info", Content: "started", EmittedAt: &emittedAt,
	}})
	if err != nil {
		t.Fatalf("batch create task progress logs failed: %v", err)
	}
	if accepted != 1 || duplicates != 0 {
		t.Fatalf("unexpected first counts accepted=%d duplicates=%d", accepted, duplicates)
	}
	accepted, duplicates, err = repo.BatchCreateTaskProgressLogs(context.Background(), []TaskProgressLogRecord{{
		ID: 1, ScanID: 7, TaskID: taskID, RequestID: "request-1", Sequence: 1, Level: "info", Content: "started", EmittedAt: &emittedAt,
	}})
	if err != nil {
		t.Fatalf("repeat batch create task progress logs failed: %v", err)
	}
	if accepted != 0 || duplicates != 1 {
		t.Fatalf("unexpected repeat counts accepted=%d duplicates=%d", accepted, duplicates)
	}

	var row struct {
		RequestID string
	}
	if err := db.Table("task_progress_log").Select("request_id").Where("task_id = ? AND sequence = ?", taskID, 1).Scan(&row).Error; err != nil {
		t.Fatalf("query task_progress_log failed: %v", err)
	}
	if row.RequestID != "request-1" {
		t.Fatalf("expected request_id persisted, got %q", row.RequestID)
	}
}

func TestTaskProgressLogRepositoryPersistsAndListsEngineProgressLogsByScanCursor(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:task_progress_log_persist_and_list?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE scan_task (
			id INTEGER PRIMARY KEY,
			scan_id INTEGER NOT NULL,
			UNIQUE (scan_id, id)
		);
		CREATE TABLE task_progress_log (
			id INTEGER NOT NULL,
			scan_id INTEGER NOT NULL,
			task_id INTEGER NOT NULL,
			request_id TEXT NOT NULL DEFAULT '',
			sequence INTEGER NOT NULL DEFAULT 0,
			level TEXT NOT NULL DEFAULT 'info',
			content TEXT NOT NULL DEFAULT '',
			emitted_at DATETIME,
			created_at DATETIME,
			PRIMARY KEY (scan_id, id),
			FOREIGN KEY (scan_id, task_id) REFERENCES scan_task(scan_id, id) ON DELETE CASCADE
		);
		CREATE UNIQUE INDEX uniq_task_progress_log_request ON task_progress_log(scan_id, task_id, request_id, sequence);
	`).Error; err != nil {
		t.Fatalf("setup task_progress_log table failed: %v", err)
	}

	taskID := 101
	otherTaskID := 102
	emittedAt := time.Date(2026, 6, 18, 10, 30, 0, 0, time.UTC)
	if err := db.Exec("INSERT INTO scan_task (id, scan_id) VALUES (?, ?), (?, ?)", taskID, 7, otherTaskID, 8).Error; err != nil {
		t.Fatalf("insert scan tasks failed: %v", err)
	}
	repo := NewTaskProgressLogRepository(db)
	accepted, duplicates, err := repo.BatchCreateTaskProgressLogs(context.Background(), []TaskProgressLogRecord{
		{ID: 1, ScanID: 7, TaskID: taskID, RequestID: "request-1", Sequence: 1, Level: "info", Content: "runtime: start", EmittedAt: &emittedAt},
		{ID: 2, ScanID: 7, TaskID: taskID, RequestID: "request-1", Sequence: 2, Level: "warning", Content: "runtime: retry", EmittedAt: ptrTime(emittedAt.Add(time.Second))},
		{ID: 1, ScanID: 8, TaskID: otherTaskID, RequestID: "request-2", Sequence: 1, Level: "info", Content: "other scan", EmittedAt: ptrTime(emittedAt.Add(2 * time.Second))},
	})
	if err != nil {
		t.Fatalf("batch create task progress logs failed: %v", err)
	}
	if accepted != 3 || duplicates != 0 {
		t.Fatalf("unexpected write counts accepted=%d duplicates=%d", accepted, duplicates)
	}

	firstPage, err := repo.FindByScanIDWithCursor(7, 0, 1)
	if err != nil {
		t.Fatalf("list first page failed: %v", err)
	}
	if len(firstPage) != 1 || firstPage[0].Content != "runtime: start" {
		t.Fatalf("unexpected first page: %+v", firstPage)
	}
	if firstPage[0].TaskID != taskID || firstPage[0].RequestID != "request-1" || firstPage[0].Sequence != 1 {
		t.Fatalf("unexpected first progress identity: %+v", firstPage[0])
	}

	nextPage, err := repo.FindByScanIDWithCursor(7, firstPage[0].ID, 10)
	if err != nil {
		t.Fatalf("list next page failed: %v", err)
	}
	if len(nextPage) != 1 || nextPage[0].Content != "runtime: retry" || nextPage[0].Level != "warning" {
		t.Fatalf("unexpected next page: %+v", nextPage)
	}
}

func TestTaskProgressLogRepositoryRejectsTasklessRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:task_progress_log_required_task?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE scan_task (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL, UNIQUE (scan_id, id));
		CREATE TABLE task_progress_log (
			id INTEGER NOT NULL,
			scan_id INTEGER NOT NULL,
			task_id INTEGER NOT NULL,
			request_id TEXT NOT NULL DEFAULT '',
			sequence INTEGER NOT NULL DEFAULT 0,
			level TEXT NOT NULL DEFAULT 'info',
			content TEXT NOT NULL DEFAULT '',
			emitted_at DATETIME,
			created_at DATETIME,
			PRIMARY KEY (scan_id, id),
			FOREIGN KEY (scan_id, task_id) REFERENCES scan_task(scan_id, id) ON DELETE CASCADE
		);
	`).Error; err != nil {
		t.Fatalf("setup task progress log schema failed: %v", err)
	}
	if err := db.Exec("INSERT INTO task_progress_log (task_id, content) VALUES (NULL, 'invalid')").Error; err == nil {
		t.Fatal("task progress log without task_id must be rejected")
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
