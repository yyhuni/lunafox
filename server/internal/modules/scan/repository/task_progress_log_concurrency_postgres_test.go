package repository

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const taskProgressLogConcurrencySchema = "task_runtime_log_concurrency"

type runtimeLogWriterContextKey struct{}

// PostgreSQL sequences allocate before commit. This test holds an inserted row
// uncommitted while another producer writes to the same Scan, reproducing the
// cursor window that SQLite's single writer cannot exercise.
func TestTaskProgressLogPostgresConcurrentWritersPreserveCursor(t *testing.T) {
	db := openTaskProgressLogConcurrencyPostgres(t)
	repo := NewTaskProgressLogRepository(db)
	inserted := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	if err := db.Callback().Create().After("gorm:create").Register("test:hold_runtime_log_commit", func(tx *gorm.DB) {
		if tx.Statement.Context.Value(runtimeLogWriterContextKey{}) != "agent" {
			return
		}
		close(inserted)
		select {
		case <-release:
		case <-tx.Statement.Context.Done():
			_ = tx.AddError(tx.Statement.Context.Err())
		}
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	first := make(chan error, 1)
	go func() {
		_, _, err := repo.BatchCreateTaskProgressLogs(context.WithValue(ctx, runtimeLogWriterContextKey{}, "agent"), runtimeLogConcurrencyRecords(7, 101, "agent", "[agent] first"))
		first <- err
	}()
	select {
	case <-inserted:
	case <-ctx.Done():
		t.Fatal("first producer did not reach uncommitted insert")
	}
	second := make(chan error, 1)
	go func() {
		_, _, err := repo.BatchCreateTaskProgressLogs(ctx, runtimeLogConcurrencyRecords(7, 101, "server", "[server] second"))
		second <- err
	}()
	awaitTaskProgressLogAdvisoryWait(t, ctx, db, second)
	// An unrelated Scan remains writable while Scan 7 waits for commit.
	if accepted, _, err := repo.BatchCreateTaskProgressLogs(ctx, runtimeLogConcurrencyRecords(8, 102, "other", "other scan")); err != nil || accepted != 1 {
		t.Fatalf("unrelated Scan was blocked by log serialization: %d %v", accepted, err)
	}
	if rows, err := repo.FindByScanIDWithCursor(7, 0, 10); err != nil || len(rows) != 0 {
		t.Fatalf("higher-ID producer became visible before first commit: %#v %v", rows, err)
	}
	releaseOnce.Do(func() { close(release) })
	for _, done := range []chan error{first, second} {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-ctx.Done():
			t.Fatal("producer did not finish after lock release")
		}
	}
	page, err := repo.FindByScanIDWithCursor(7, 0, 1)
	if err != nil || len(page) != 1 || page[0].Content != "[agent] first" {
		t.Fatalf("first cursor page = %#v %v", page, err)
	}
	next, err := repo.FindByScanIDWithCursor(7, page[0].ID, 10)
	if err != nil || len(next) != 1 || next[0].Content != "[server] second" || next[0].ID <= page[0].ID {
		t.Fatalf("cursor skipped concurrent producer: %#v %v", next, err)
	}
	if accepted, duplicates, err := repo.BatchCreateTaskProgressLogs(ctx, runtimeLogConcurrencyRecords(7, 101, "server", "[server] second")); err != nil || accepted != 0 || duplicates != 1 {
		t.Fatalf("serialization changed replay deduplication: %d %d %v", accepted, duplicates, err)
	}
}

func TestTaskProgressLogPostgresLockWaitHonorsEvidenceDeadline(t *testing.T) {
	db := openTaskProgressLogConcurrencyPostgres(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	if err := tx.Exec("SELECT pg_advisory_xact_lock(?, ?)", taskProgressLogAdvisoryLockNamespace, 7).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	accepted, _, err := NewTaskProgressLogRepository(db).BatchCreateTaskProgressLogs(ctx, runtimeLogConcurrencyRecords(7, 101, "deadline", "cancelled"))
	if err == nil || accepted != 0 || !errors.Is(ctx.Err(), context.DeadlineExceeded) || time.Since(started) > time.Second {
		t.Fatalf("log lock ignored evidence deadline: %d %v", accepted, err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatal(err)
	}
	if rows, err := NewTaskProgressLogRepository(db).FindByScanIDWithCursor(7, 0, 10); err != nil || len(rows) != 0 {
		t.Fatalf("cancelled lock wait persisted evidence: %#v %v", rows, err)
	}
}

func runtimeLogConcurrencyRecords(scanID, taskID int, requestID, content string) []TaskProgressLogRecord {
	emittedAt := time.Now().UTC()
	return []TaskProgressLogRecord{{ScanID: scanID, TaskID: taskID, RequestID: requestID, Sequence: 1, Level: "info", Content: content, EmittedAt: &emittedAt}}
}

func awaitTaskProgressLogAdvisoryWait(t *testing.T, ctx context.Context, db *gorm.DB, done <-chan error) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var waiting int64
		if err := db.WithContext(ctx).Raw("SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock' AND wait_event = 'advisory'").Scan(&waiting).Error; err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			return
		}
		select {
		case err := <-done:
			t.Fatalf("concurrent producer bypassed same-Scan commit serialization: %v", err)
		case <-ctx.Done():
			t.Fatal("concurrent producer did not wait on the same-Scan advisory lock")
		case <-ticker.C:
		}
	}
}

func openTaskProgressLogConcurrencyPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("LUNAFOX_POSTGRES_RUNTIME_LOG_DSN"))
	if dsn == "" {
		t.Skip("set LUNAFOX_POSTGRES_RUNTIME_LOG_DSN to run PostgreSQL runtime-log commit ordering verification")
	}
	if !strings.Contains(dsn, "search_path="+taskProgressLogConcurrencySchema) {
		t.Fatalf("runtime-log DSN must pin search_path=%s", taskProgressLogConcurrencySchema)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("DROP SCHEMA IF EXISTS " + taskProgressLogConcurrencySchema + " CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE SCHEMA " + taskProgressLogConcurrencySchema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = db.WithContext(cleanupCtx).Exec("DROP SCHEMA IF EXISTS " + taskProgressLogConcurrencySchema + " CASCADE").Error
	})
	statements := []string{
		"CREATE TABLE scan_task (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL, UNIQUE (scan_id, id))",
		"CREATE TABLE task_progress_log (id BIGSERIAL PRIMARY KEY, scan_id INTEGER NOT NULL, task_id INTEGER NOT NULL, request_id TEXT NOT NULL, sequence BIGINT NOT NULL, level TEXT NOT NULL, content TEXT NOT NULL, emitted_at TIMESTAMPTZ, created_at TIMESTAMPTZ, UNIQUE (scan_id, task_id, request_id, sequence), FOREIGN KEY (scan_id, task_id) REFERENCES scan_task(scan_id, id) ON DELETE CASCADE)",
		"INSERT INTO scan_task (id, scan_id) VALUES (101, 7), (102, 8)",
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}
