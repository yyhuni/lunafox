package application

import (
	"context"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type screenshotSnapshotQueryStoreStub struct {
	items    []snapshotdomain.ScreenshotSnapshot
	total    int64
	itemByID map[int]*snapshotdomain.ScreenshotSnapshot
	listErr  error
	findErr  error
}

func (stub *screenshotSnapshotQueryStoreStub) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.ScreenshotSnapshot, int64, error) {
	_ = scanID
	_ = page
	_ = pageSize
	_ = filter
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]snapshotdomain.ScreenshotSnapshot(nil), stub.items...), stub.total, nil
}

func (stub *screenshotSnapshotQueryStoreStub) FindByIDAndScanID(id int, scanID int) (*snapshotdomain.ScreenshotSnapshot, error) {
	_ = scanID
	if stub.findErr != nil {
		return nil, stub.findErr
	}
	item, ok := stub.itemByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyItem := *item
	return &copyItem, nil
}

func TestScreenshotSnapshotQueryServiceListAndGetByID(t *testing.T) {
	store := &screenshotSnapshotQueryStoreStub{
		items:    []snapshotdomain.ScreenshotSnapshot{{ID: 1}},
		total:    1,
		itemByID: map[int]*snapshotdomain.ScreenshotSnapshot{2: {ID: 2}},
	}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 6}}
	service := NewScreenshotSnapshotQueryService(store, lookup)

	items, total, err := service.ListByScan(context.Background(), 6, 1, 20, "")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 || total != 1 {
		t.Fatalf("unexpected list result len=%d total=%d", len(items), total)
	}

	item, err := service.GetByID(context.Background(), 6, 2)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if item.ID != 2 {
		t.Fatalf("unexpected item id=%d", item.ID)
	}
}

func TestScreenshotSnapshotQueryServiceErrors(t *testing.T) {
	service := NewScreenshotSnapshotQueryService(&screenshotSnapshotQueryStoreStub{itemByID: map[int]*snapshotdomain.ScreenshotSnapshot{}}, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound})

	_, _, err := service.ListByScan(context.Background(), 1, 1, 20, "")
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}

	service = NewScreenshotSnapshotQueryService(&screenshotSnapshotQueryStoreStub{itemByID: map[int]*snapshotdomain.ScreenshotSnapshot{}}, &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1}})
	_, err = service.GetByID(context.Background(), 1, 99)
	if !errors.Is(err, ErrScreenshotSnapshotNotFound) {
		t.Fatalf("expected ErrScreenshotSnapshotNotFound, got %v", err)
	}
}
