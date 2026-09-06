package repository

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDirectorySnapshotBatchUpsertReplacesCompleteObservationAndReplayConverges(t *testing.T) {
	db := newDirectorySnapshotRepositoryDB(t)
	repo := NewDirectorySnapshotRepository(db)
	url := "https://Example.com/%00?x=1#frag"
	firstStatus, secondStatus := 999, 0
	firstLength, secondLength := int64(math.MaxInt64), int64(0)
	firstDuration, secondDuration := int64(math.MaxInt64), int64(0)

	first := snapshotdomain.DirectorySnapshot{
		ScanID: 7, URL: url, Status: &firstStatus, ContentLength: &firstLength,
		ContentType: "application/octet-stream", Duration: &firstDuration,
	}
	if affected, err := repo.BatchCreate([]snapshotdomain.DirectorySnapshot{first}); err != nil || affected != 1 {
		t.Fatalf("first Directory Snapshot upsert = affected %d, err %v", affected, err)
	}

	var created model.DirectorySnapshot
	if err := db.Where("scan_id = ? AND url = ?", 7, url).First(&created).Error; err != nil {
		t.Fatalf("read first Directory Snapshot: %v", err)
	}
	if created.ContentLength == nil || *created.ContentLength != math.MaxInt64 || created.Duration == nil || *created.Duration != math.MaxInt64 {
		t.Fatalf("MaxInt64 Snapshot observation was not persisted exactly: %+v", created)
	}

	replacement := snapshotdomain.DirectorySnapshot{
		ScanID: 7, URL: url, Status: &secondStatus, ContentLength: &secondLength,
		ContentType: "", Duration: &secondDuration,
	}
	for attempt := 0; attempt < 2; attempt++ {
		if affected, err := repo.BatchCreate([]snapshotdomain.DirectorySnapshot{replacement}); err != nil || affected != 1 {
			t.Fatalf("Snapshot replacement attempt %d = affected %d, err %v", attempt+1, affected, err)
		}
	}

	var stored model.DirectorySnapshot
	if err := db.Where("scan_id = ? AND url = ?", 7, url).First(&stored).Error; err != nil {
		t.Fatalf("read replaced Directory Snapshot: %v", err)
	}
	if stored.ID != created.ID || !stored.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("Snapshot upsert changed stable identity fields: before=%+v after=%+v", created, stored)
	}
	if stored.Status == nil || *stored.Status != 0 || stored.ContentLength == nil || *stored.ContentLength != 0 || stored.ContentType != "" || stored.Duration == nil || *stored.Duration != 0 {
		t.Fatalf("Snapshot replacement did not use one complete zero-valued observation: %+v", stored)
	}
	var count int64
	if err := db.Model(&model.DirectorySnapshot{}).Where("scan_id = ? AND url = ?", 7, url).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("Snapshot replay did not converge to one row: count=%d err=%v", count, err)
	}
}

func TestDirectorySnapshotRepositoryKeepsExactURLIdentitiesVisibleForFailedScan(t *testing.T) {
	db := newDirectorySnapshotRepositoryDB(t)
	repo := NewDirectorySnapshotRepository(db)
	status := 200
	length, duration := int64(1), int64(2)
	items := []snapshotdomain.DirectorySnapshot{
		{ScanID: 7, URL: "https://Example.com/%00", Status: &status, ContentLength: &length, ContentType: "text/html", Duration: &duration},
		{ScanID: 7, URL: "https://example.com/%00", Status: &status, ContentLength: &length, ContentType: "text/html", Duration: &duration},
	}
	if affected, err := repo.BatchCreate(items); err != nil || affected != 2 {
		t.Fatalf("exact-identity Snapshot upsert = affected %d, err %v", affected, err)
	}

	listed, total, err := repo.ListByScanID(7, 1, 20, "", "createdAt asc")
	if err != nil || total != 2 || len(listed) != 2 {
		t.Fatalf("failed Scan hid Directory Snapshots: total=%d listed=%d err=%v", total, len(listed), err)
	}
	seen := map[string]bool{}
	if err := repo.ForEachByScanID(context.Background(), 7, func(item snapshotdomain.DirectorySnapshot) error {
		seen[item.URL] = true
		return nil
	}); err != nil {
		t.Fatalf("stream Directory Snapshots: %v", err)
	}
	if !seen[items[0].URL] || !seen[items[1].URL] {
		t.Fatalf("exact Snapshot identities were collapsed or hidden: %#v", seen)
	}
	if count, err := repo.CountByScanID(7); err != nil || count != 2 {
		t.Fatalf("Directory Snapshot count for failed Scan = %d, err %v", count, err)
	}
}

func newDirectorySnapshotRepositoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open Directory Snapshot sqlite db: %v", err)
	}
	for _, statement := range []string{
		`CREATE TABLE scan (id INTEGER PRIMARY KEY, target_id INTEGER NOT NULL, status TEXT NOT NULL)`,
		`INSERT INTO scan (id, target_id, status) VALUES (7, 1, 'failed')`,
		`CREATE TABLE directory_snapshot (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			url TEXT NOT NULL,
			status INTEGER,
			content_length INTEGER,
			content_type TEXT NOT NULL DEFAULT '',
			duration INTEGER,
			created_at DATETIME,
			UNIQUE(scan_id, url)
		)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare Directory Snapshot schema: %v", err)
		}
	}
	return db
}
