package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type subdomainSnapshotQueryStoreStub struct {
	items     []snapshotdomain.SubdomainSnapshot
	total     int64
	count     int64
	listErr   error
	streamErr error
	countErr  error
	scanErr   error
}

func (stub *subdomainSnapshotQueryStoreStub) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.SubdomainSnapshot, int64, error) {
	_ = scanID
	_ = page
	_ = pageSize
	_ = filter
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]snapshotdomain.SubdomainSnapshot(nil), stub.items...), stub.total, nil
}

func (stub *subdomainSnapshotQueryStoreStub) StreamByScanID(scanID int) (*sql.Rows, error) {
	_ = scanID
	if stub.streamErr != nil {
		return nil, stub.streamErr
	}
	return nil, nil
}

func (stub *subdomainSnapshotQueryStoreStub) CountByScanID(scanID int) (int64, error) {
	_ = scanID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func (stub *subdomainSnapshotQueryStoreStub) ScanRow(rows *sql.Rows) (*snapshotdomain.SubdomainSnapshot, error) {
	_ = rows
	if stub.scanErr != nil {
		return nil, stub.scanErr
	}
	return &snapshotdomain.SubdomainSnapshot{ID: 1}, nil
}

func TestSubdomainSnapshotQueryServiceListAndCount(t *testing.T) {
	store := &subdomainSnapshotQueryStoreStub{items: []snapshotdomain.SubdomainSnapshot{{ID: 1}}, total: 1, count: 4}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 3}}
	service := NewSubdomainSnapshotQueryService(store, lookup)

	items, total, err := service.ListByScan(context.Background(), 3, 1, 20, "")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 || total != 1 {
		t.Fatalf("unexpected list result len=%d total=%d", len(items), total)
	}

	count, err := service.CountByScan(context.Background(), 3)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected count 4, got %d", count)
	}
}

func TestSubdomainSnapshotQueryServiceScanNotFound(t *testing.T) {
	store := &subdomainSnapshotQueryStoreStub{}
	lookup := &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}
	service := NewSubdomainSnapshotQueryService(store, lookup)

	_, _, err := service.ListByScan(context.Background(), 1, 1, 20, "")
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}
}
