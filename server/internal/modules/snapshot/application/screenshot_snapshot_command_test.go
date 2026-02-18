package application

import (
	"context"
	"errors"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
	"testing"
)

type screenshotSnapshotCommandStoreStub struct {
	snapshots []snapshotdomain.ScreenshotSnapshot
	err       error
}

func (stub *screenshotSnapshotCommandStoreStub) BulkUpsert(snapshots []snapshotdomain.ScreenshotSnapshot) (int64, error) {
	if stub.err != nil {
		return 0, stub.err
	}
	stub.snapshots = append([]snapshotdomain.ScreenshotSnapshot(nil), snapshots...)
	return int64(len(snapshots)), nil
}

type screenshotAssetSyncStub struct {
	items []ScreenshotAssetItem
	err   error
}

func (stub *screenshotAssetSyncStub) BulkUpsert(targetID int, req *ScreenshotAssetUpsertRequest) (int64, error) {
	_ = targetID
	if stub.err != nil {
		return 0, stub.err
	}
	stub.items = append([]ScreenshotAssetItem(nil), req.Screenshots...)
	return int64(len(req.Screenshots)), nil
}

func TestScreenshotSnapshotCommandServiceSaveAndSync(t *testing.T) {
	store := &screenshotSnapshotCommandStoreStub{}
	assetSync := &screenshotAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 4, TargetID: 6},
		target: &snapshotdomain.ScanTargetRef{ID: 6, Name: "example.com", Type: "domain"},
	}
	service := NewScreenshotSnapshotCommandService(store, lookup, assetSync)

	snapshotCount, assetCount, err := service.SaveAndSync(context.Background(), 4, 6, []ScreenshotSnapshotItem{
		{URL: "https://example.com/a"},
		{URL: "https://evil.com/a"},
	})
	if err != nil {
		t.Fatalf("save and sync failed: %v", err)
	}
	if snapshotCount != 1 || assetCount != 1 {
		t.Fatalf("unexpected counts snapshot=%d asset=%d", snapshotCount, assetCount)
	}
	if len(store.snapshots) != 1 || len(assetSync.items) != 1 {
		t.Fatalf("unexpected stored counts snapshots=%d items=%d", len(store.snapshots), len(assetSync.items))
	}
}

func TestScreenshotSnapshotCommandServiceErrors(t *testing.T) {
	service := NewScreenshotSnapshotCommandService(&screenshotSnapshotCommandStoreStub{}, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}, &screenshotAssetSyncStub{})

	_, _, err := service.SaveAndSync(context.Background(), 1, 1, []ScreenshotSnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}

	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1, TargetID: 9}, target: &snapshotdomain.ScanTargetRef{ID: 9, Name: "example.com", Type: "domain"}}
	service = NewScreenshotSnapshotCommandService(&screenshotSnapshotCommandStoreStub{}, lookup, &screenshotAssetSyncStub{})
	_, _, err = service.SaveAndSync(context.Background(), 1, 8, []ScreenshotSnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotTargetMismatch) {
		t.Fatalf("expected ErrSnapshotTargetMismatch, got %v", err)
	}
}
