package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type websiteSnapshotQueryStoreStub struct {
	items      []snapshotdomain.WebsiteSnapshot
	total      int64
	count      int64
	listErr    error
	streamErr  error
	countErr   error
	scanRowErr error
}

func (stub *websiteSnapshotQueryStoreStub) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.WebsiteSnapshot, int64, error) {
	_ = scanID
	_ = page
	_ = pageSize
	_ = filter
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]snapshotdomain.WebsiteSnapshot(nil), stub.items...), stub.total, nil
}

func (stub *websiteSnapshotQueryStoreStub) StreamByScanID(scanID int) (*sql.Rows, error) {
	_ = scanID
	if stub.streamErr != nil {
		return nil, stub.streamErr
	}
	return nil, nil
}

func (stub *websiteSnapshotQueryStoreStub) CountByScanID(scanID int) (int64, error) {
	_ = scanID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func (stub *websiteSnapshotQueryStoreStub) ScanRow(rows *sql.Rows) (*snapshotdomain.WebsiteSnapshot, error) {
	_ = rows
	if stub.scanRowErr != nil {
		return nil, stub.scanRowErr
	}
	return &snapshotdomain.WebsiteSnapshot{ID: 1}, nil
}

type snapshotScanLookupStub struct {
	scan      *snapshotdomain.ScanRef
	target    *snapshotdomain.ScanTargetRef
	findErr   error
	targetErr error
}

func (stub *snapshotScanLookupStub) GetScanRefByID(id int) (*snapshotdomain.ScanRef, error) {
	_ = id
	if stub.findErr != nil {
		return nil, stub.findErr
	}
	if stub.scan == nil {
		return nil, gorm.ErrRecordNotFound
	}
	copyScan := *stub.scan
	return &copyScan, nil
}

func (stub *snapshotScanLookupStub) GetTargetRefByScanID(scanID int) (*snapshotdomain.ScanTargetRef, error) {
	_ = scanID
	if stub.targetErr != nil {
		return nil, stub.targetErr
	}
	if stub.target == nil {
		return nil, gorm.ErrRecordNotFound
	}
	copyTarget := *stub.target
	return &copyTarget, nil
}

func TestWebsiteSnapshotQueryServiceListAndCount(t *testing.T) {
	store := &websiteSnapshotQueryStoreStub{
		items: []snapshotdomain.WebsiteSnapshot{{ID: 1}, {ID: 2}},
		total: 2,
		count: 9,
	}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 7}}
	service := NewWebsiteSnapshotQueryService(store, lookup)

	items, total, err := service.ListByScan(context.Background(), 7, 1, 20, "")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 2 || total != 2 {
		t.Fatalf("unexpected list result len=%d total=%d", len(items), total)
	}

	count, err := service.CountByScan(context.Background(), 7)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 9 {
		t.Fatalf("expected count 9, got %d", count)
	}
}

func TestWebsiteSnapshotQueryServiceScanNotFound(t *testing.T) {
	store := &websiteSnapshotQueryStoreStub{}
	lookup := &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}
	service := NewWebsiteSnapshotQueryService(store, lookup)

	_, _, err := service.ListByScan(context.Background(), 1, 1, 20, "")
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}
}
