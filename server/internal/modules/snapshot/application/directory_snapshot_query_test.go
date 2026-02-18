package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type directorySnapshotQueryStoreStub struct {
	items     []snapshotdomain.DirectorySnapshot
	total     int64
	count     int64
	listErr   error
	streamErr error
	countErr  error
	scanErr   error
}

func (stub *directorySnapshotQueryStoreStub) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.DirectorySnapshot, int64, error) {
	_ = scanID
	_ = page
	_ = pageSize
	_ = filter
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]snapshotdomain.DirectorySnapshot(nil), stub.items...), stub.total, nil
}

func (stub *directorySnapshotQueryStoreStub) StreamByScanID(scanID int) (*sql.Rows, error) {
	_ = scanID
	if stub.streamErr != nil {
		return nil, stub.streamErr
	}
	return nil, nil
}

func (stub *directorySnapshotQueryStoreStub) CountByScanID(scanID int) (int64, error) {
	_ = scanID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func (stub *directorySnapshotQueryStoreStub) ScanRow(rows *sql.Rows) (*snapshotdomain.DirectorySnapshot, error) {
	_ = rows
	if stub.scanErr != nil {
		return nil, stub.scanErr
	}
	return &snapshotdomain.DirectorySnapshot{ID: 1}, nil
}

func TestDirectorySnapshotQueryServiceListAndCount(t *testing.T) {
	store := &directorySnapshotQueryStoreStub{items: []snapshotdomain.DirectorySnapshot{{ID: 1}}, total: 1, count: 3}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 9}}
	service := NewDirectorySnapshotQueryService(store, lookup)

	items, total, err := service.ListByScan(context.Background(), 9, 1, 20, "")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 || total != 1 {
		t.Fatalf("unexpected list result len=%d total=%d", len(items), total)
	}

	count, err := service.CountByScan(context.Background(), 9)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}
}

func TestDirectorySnapshotQueryServiceScanNotFound(t *testing.T) {
	store := &directorySnapshotQueryStoreStub{}
	lookup := &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}
	service := NewDirectorySnapshotQueryService(store, lookup)

	_, _, err := service.ListByScan(context.Background(), 1, 1, 20, "")
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}
}
