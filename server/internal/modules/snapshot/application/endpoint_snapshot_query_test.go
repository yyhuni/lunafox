package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type endpointSnapshotQueryStoreStub struct {
	items     []snapshotdomain.EndpointSnapshot
	total     int64
	count     int64
	listErr   error
	streamErr error
	countErr  error
	scanErr   error
}

func (stub *endpointSnapshotQueryStoreStub) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.EndpointSnapshot, int64, error) {
	_ = scanID
	_ = page
	_ = pageSize
	_ = filter
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]snapshotdomain.EndpointSnapshot(nil), stub.items...), stub.total, nil
}

func (stub *endpointSnapshotQueryStoreStub) StreamByScanID(scanID int) (*sql.Rows, error) {
	_ = scanID
	if stub.streamErr != nil {
		return nil, stub.streamErr
	}
	return nil, nil
}

func (stub *endpointSnapshotQueryStoreStub) CountByScanID(scanID int) (int64, error) {
	_ = scanID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func (stub *endpointSnapshotQueryStoreStub) ScanRow(rows *sql.Rows) (*snapshotdomain.EndpointSnapshot, error) {
	_ = rows
	if stub.scanErr != nil {
		return nil, stub.scanErr
	}
	return &snapshotdomain.EndpointSnapshot{ID: 1}, nil
}

func TestEndpointSnapshotQueryServiceListAndCount(t *testing.T) {
	store := &endpointSnapshotQueryStoreStub{items: []snapshotdomain.EndpointSnapshot{{ID: 1}}, total: 1, count: 5}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 6}}
	service := NewEndpointSnapshotQueryService(store, lookup)

	items, total, err := service.ListByScan(context.Background(), 6, 1, 20, "")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 || total != 1 {
		t.Fatalf("unexpected list result len=%d total=%d", len(items), total)
	}

	count, err := service.CountByScan(context.Background(), 6)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected count 5, got %d", count)
	}
}

func TestEndpointSnapshotQueryServiceScanNotFound(t *testing.T) {
	store := &endpointSnapshotQueryStoreStub{}
	lookup := &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}
	service := NewEndpointSnapshotQueryService(store, lookup)

	_, _, err := service.ListByScan(context.Background(), 1, 1, 20, "")
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}
}
