package repository

import (
	"context"
	"strings"
	"testing"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type runtimeLogScanLookup struct{}

func (runtimeLogScanLookup) GetTaskProgressLogRefByID(id int) (*scanapp.TaskProgressLogScanRef, error) {
	return &scanapp.TaskProgressLogScanRef{ID: id}, nil
}

func TestTaskRuntimeLogAppenderPersistsRawChunksAndDeduplicatesReplay(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE scan_task (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL, UNIQUE (scan_id, id));
		CREATE TABLE task_progress_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			task_id INTEGER NOT NULL,
			request_id TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			level TEXT NOT NULL,
			content TEXT NOT NULL,
			emitted_at DATETIME,
			created_at DATETIME,
			UNIQUE (scan_id, task_id, request_id, sequence),
			FOREIGN KEY (scan_id, task_id) REFERENCES scan_task(scan_id, id) ON DELETE CASCADE
		);
		INSERT INTO scan_task (id, scan_id) VALUES (101, 7);
	`).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewTaskProgressLogRepository(db)
	service := scanapp.NewTaskProgressLogService(repo, repo, runtimeLogScanLookup{})
	appender := scanapp.NewTaskRuntimeLogAppender(service)
	raw := "[server] original DB error /srv/private token=raw-canary\n\x1b[31m" + strings.Repeat("诊断", 4000)
	event := scanapp.TaskRuntimeLogEvent{ScanID: 7, TaskID: 101, EventID: "result:batch-1", Level: "error", Content: raw}
	appender.AppendTaskRuntimeLog(context.Background(), event)
	appender.AppendTaskRuntimeLog(context.Background(), event)
	rows, err := repo.FindByScanIDWithCursor(7, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].ID >= rows[1].ID || rows[0].Sequence != 1 || rows[1].Sequence != 2 || rows[0].RequestID != rows[1].RequestID {
		t.Fatalf("replay persisted duplicate or unordered chunks: %#v", rows)
	}
	if rows[0].Content+rows[1].Content != raw {
		t.Fatal("raw Server evidence changed across persistence")
	}
	// The original task foreign key and scan-history lifecycle keep owning the
	// evidence; the new appender does not create an independent retained store.
	if err := db.Exec("DELETE FROM scan_task WHERE id = 101").Error; err != nil {
		t.Fatal(err)
	}
	rows, err = repo.FindByScanIDWithCursor(7, 0, 10)
	if err != nil || len(rows) != 0 {
		t.Fatalf("task-owned logs survived their owner: %#v %v", rows, err)
	}
}
