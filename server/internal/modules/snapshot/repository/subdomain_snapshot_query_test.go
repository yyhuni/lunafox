package repository

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSubdomainSnapshotRepositoryForEachDNSNameByScanIDStreamsCanonicalNames(t *testing.T) {
	db := newSubdomainSnapshotRepositoryDB(t)
	createdAt := time.Date(2026, 7, 1, 10, 30, 0, 0, time.UTC)
	if err := db.Exec(`
		INSERT INTO subdomain_snapshot (scan_id, dns_name, created_at) VALUES
			(10, 'beta.example.com', ?),
			(10, 'api.example.com', ?),
			(10, '', ?),
			(11, 'zzz.example.com', ?)
	`, createdAt, createdAt, createdAt, createdAt).Error; err != nil {
		t.Fatalf("insert subdomain snapshots failed: %v", err)
	}

	var got []string
	err := NewSubdomainSnapshotRepository(db).ForEachDNSNameByScanID(context.Background(), 10, func(name string) error {
		got = append(got, name)
		return nil
	})
	if err != nil {
		t.Fatalf("ForEachDNSNameByScanID returned error: %v", err)
	}
	want := []string{"", "api.example.com", "beta.example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected DNS names: got %#v want %#v", got, want)
	}
}

func TestSubdomainSnapshotRepositoryForEachDNSNameByScanIDPropagatesVisitError(t *testing.T) {
	db := newSubdomainSnapshotRepositoryDB(t)
	if err := db.Exec(`
		INSERT INTO subdomain_snapshot (scan_id, dns_name, created_at) VALUES
			(10, 'api.example.com', CURRENT_TIMESTAMP),
			(10, 'www.example.com', CURRENT_TIMESTAMP)
	`).Error; err != nil {
		t.Fatalf("insert subdomain snapshots failed: %v", err)
	}

	sentinel := errors.New("stop stream")
	visited := 0
	err := NewSubdomainSnapshotRepository(db).ForEachDNSNameByScanID(context.Background(), 10, func(name string) error {
		visited++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected visitor error, got %v", err)
	}
	if visited != 1 {
		t.Fatalf("expected stream to stop after visitor error, visited %d", visited)
	}
}

func TestSubdomainSnapshotRepositoryListByScanIDUsesScopedStableOrdering(t *testing.T) {
	db := newSubdomainSnapshotRepositoryDB(t)
	older := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	if err := db.Exec(`
		INSERT INTO subdomain_snapshot (id, scan_id, dns_name, created_at) VALUES
			(1, 10, 'b.example.com', ?),
			(2, 10, 'a.example.com', ?),
			(3, 10, 'a.example.com', ?),
			(4, 11, 'z.example.com', ?)
	`, older, newer, newer, newer).Error; err != nil {
		t.Fatalf("insert subdomain snapshots failed: %v", err)
	}

	items, total, err := NewSubdomainSnapshotRepository(db).ListByScanID(10, 1, 10, "", "dnsName asc")
	if err != nil {
		t.Fatalf("ListByScanID returned error: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected scan-scoped total 3, got %d", total)
	}
	gotIDs := []int{items[0].ID, items[1].ID, items[2].ID}
	wantIDs := []int{2, 3, 1}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("unexpected dnsName asc stable order: got %v want %v", gotIDs, wantIDs)
	}

	items, _, err = NewSubdomainSnapshotRepository(db).ListByScanID(10, 1, 10, "", "createdAt desc")
	if err != nil {
		t.Fatalf("ListByScanID returned error: %v", err)
	}
	gotIDs = []int{items[0].ID, items[1].ID, items[2].ID}
	wantIDs = []int{3, 2, 1}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("unexpected createdAt desc stable order: got %v want %v", gotIDs, wantIDs)
	}
}

func newSubdomainSnapshotRepositoryDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE subdomain_snapshot (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			dns_name TEXT,
			created_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("setup subdomain_snapshot table failed: %v", err)
	}
	return db
}
