package repository

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHostPortSnapshotRepositoryStreamsCompleteFactsInStableOrder(t *testing.T) {
	db := newHostPortSnapshotCursorDB(t)
	if err := db.Exec(`
		INSERT INTO host_port_mapping_snapshot (scan_id, host, ip, port) VALUES
			(10, 'API.Example.com', '192.0.2.1', 443),
			(10, 'api.example.com', '192.0.2.2', 443),
			(10, '192.0.2.3', '192.0.2.3', 80),
			(10, 'www.example.com', '192.0.2.4', 8080),
			(11, 'other.example.com', '192.0.2.5', 80)
	`).Error; err != nil {
		t.Fatalf("insert HostPort snapshots: %v", err)
	}

	var got []snapshotdomain.HostPortInputEvidence
	err := NewHostPortSnapshotRepository(db).ForEachHostPortByScanID(context.Background(), 10, func(evidence snapshotdomain.HostPortInputEvidence) error {
		got = append(got, evidence)
		return nil
	})
	if err != nil {
		t.Fatalf("ForEachHostPortByScanID returned error: %v", err)
	}
	want := []snapshotdomain.HostPortInputEvidence{
		{Host: "192.0.2.3", IP: "192.0.2.3", Port: 80},
		{Host: "API.Example.com", IP: "192.0.2.1", Port: 443},
		{Host: "api.example.com", IP: "192.0.2.2", Port: 443},
		{Host: "www.example.com", IP: "192.0.2.4", Port: 8080},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("evidence = %#v, want %#v", got, want)
	}
}

func TestHostPortSnapshotRepositoryCursorStopsOnVisitorError(t *testing.T) {
	db := newHostPortSnapshotCursorDB(t)
	if err := db.Exec(`INSERT INTO host_port_mapping_snapshot (scan_id, host, ip, port) VALUES (10, 'api.example.com', '192.0.2.1', 443)`).Error; err != nil {
		t.Fatalf("insert HostPort snapshot: %v", err)
	}
	sentinel := errors.New("stop")
	err := NewHostPortSnapshotRepository(db).ForEachHostPortByScanID(context.Background(), 10, func(snapshotdomain.HostPortInputEvidence) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("visitor error = %v, want sentinel", err)
	}
}

func newHostPortSnapshotCursorDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE host_port_mapping_snapshot (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			host TEXT NOT NULL,
			ip TEXT NOT NULL,
			port INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		t.Fatalf("create HostPort snapshot table: %v", err)
	}
	return db
}
