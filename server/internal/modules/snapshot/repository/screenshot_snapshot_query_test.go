package repository

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestScreenshotSnapshotRepositoryListByScanIDUsesScopedStableOrderingAndNullsLast(t *testing.T) {
	db := newScreenshotSnapshotRepositoryDB(t)
	older := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	if err := db.Exec(`
		INSERT INTO screenshot_snapshot (id, scan_id, url, status_code, created_at) VALUES
			(1, 10, 'https://example.com/old', 404, ?),
			(2, 10, 'https://example.com/null-status', NULL, ?),
			(3, 10, 'https://example.com/new', 200, ?),
			(4, 11, 'https://other.example.com/new', 200, ?)
	`, older, newer, newer, newer).Error; err != nil {
		t.Fatalf("insert screenshot snapshots failed: %v", err)
	}

	items, total, err := NewScreenshotSnapshotRepository(db).ListByScanID(10, 1, 10, "", "createdAt desc")
	if err != nil {
		t.Fatalf("ListByScanID returned error: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected scan-scoped total 3, got %d", total)
	}
	gotIDs := []int{items[0].ID, items[1].ID, items[2].ID}
	wantIDs := []int{3, 2, 1}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("unexpected createdAt desc stable order: got %v want %v", gotIDs, wantIDs)
	}

	items, _, err = NewScreenshotSnapshotRepository(db).ListByScanID(10, 1, 10, "", "statusCode asc")
	if err != nil {
		t.Fatalf("ListByScanID returned error: %v", err)
	}
	gotIDs = []int{items[0].ID, items[1].ID, items[2].ID}
	wantIDs = []int{3, 1, 2}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("unexpected statusCode asc nulls-last order: got %v want %v", gotIDs, wantIDs)
	}
}

func newScreenshotSnapshotRepositoryDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE screenshot_snapshot (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			url TEXT NOT NULL,
			status_code INTEGER,
			image BLOB,
			created_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("setup screenshot_snapshot table failed: %v", err)
	}
	return db
}
