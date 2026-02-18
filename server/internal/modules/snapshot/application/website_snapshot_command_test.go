package application

import (
	"context"
	"errors"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
	"testing"
)

type websiteSnapshotCommandStoreStub struct {
	snapshots []snapshotdomain.WebsiteSnapshot
	createErr error
}

func (stub *websiteSnapshotCommandStoreStub) BulkCreate(snapshots []snapshotdomain.WebsiteSnapshot) (int64, error) {
	if stub.createErr != nil {
		return 0, stub.createErr
	}
	stub.snapshots = append([]snapshotdomain.WebsiteSnapshot(nil), snapshots...)
	return int64(len(snapshots)), nil
}

type websiteAssetSyncStub struct {
	items []WebsiteAssetUpsertItem
	err   error
}

func (stub *websiteAssetSyncStub) BulkUpsert(targetID int, items []WebsiteAssetUpsertItem) (int64, error) {
	_ = targetID
	if stub.err != nil {
		return 0, stub.err
	}
	stub.items = append([]WebsiteAssetUpsertItem(nil), items...)
	return int64(len(items)), nil
}

func TestWebsiteSnapshotCommandServiceSaveAndSync(t *testing.T) {
	store := &websiteSnapshotCommandStoreStub{}
	assetSync := &websiteAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 11, TargetID: 7},
		target: &snapshotdomain.ScanTargetRef{ID: 7, Name: "example.com", Type: "domain"},
	}
	service := NewWebsiteSnapshotCommandService(store, lookup, assetSync)

	snapshotCount, assetCount, err := service.SaveAndSync(context.Background(), 11, 7, []WebsiteSnapshotItem{
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
	if store.snapshots[0].Host != "example.com" {
		t.Fatalf("expected derived host example.com, got %s", store.snapshots[0].Host)
	}
}

func TestWebsiteSnapshotCommandServiceErrors(t *testing.T) {
	service := NewWebsiteSnapshotCommandService(&websiteSnapshotCommandStoreStub{}, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}, &websiteAssetSyncStub{})

	_, _, err := service.SaveAndSync(context.Background(), 1, 1, []WebsiteSnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}

	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1, TargetID: 9}, target: &snapshotdomain.ScanTargetRef{ID: 9, Name: "example.com", Type: "domain"}}
	service = NewWebsiteSnapshotCommandService(&websiteSnapshotCommandStoreStub{}, lookup, &websiteAssetSyncStub{})
	_, _, err = service.SaveAndSync(context.Background(), 1, 8, []WebsiteSnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotTargetMismatch) {
		t.Fatalf("expected ErrSnapshotTargetMismatch, got %v", err)
	}
}
