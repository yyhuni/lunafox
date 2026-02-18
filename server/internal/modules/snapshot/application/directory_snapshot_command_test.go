package application

import (
	"context"
	"errors"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
	"testing"
)

type directorySnapshotCommandStoreStub struct {
	snapshots []snapshotdomain.DirectorySnapshot
	err       error
}

func (stub *directorySnapshotCommandStoreStub) BulkCreate(snapshots []snapshotdomain.DirectorySnapshot) (int64, error) {
	if stub.err != nil {
		return 0, stub.err
	}
	stub.snapshots = append([]snapshotdomain.DirectorySnapshot(nil), snapshots...)
	return int64(len(snapshots)), nil
}

type directoryAssetSyncStub struct {
	items []DirectoryAssetUpsertItem
	err   error
}

func (stub *directoryAssetSyncStub) BulkUpsert(targetID int, items []DirectoryAssetUpsertItem) (int64, error) {
	_ = targetID
	if stub.err != nil {
		return 0, stub.err
	}
	stub.items = append([]DirectoryAssetUpsertItem(nil), items...)
	return int64(len(items)), nil
}

func TestDirectorySnapshotCommandServiceSaveAndSync(t *testing.T) {
	store := &directorySnapshotCommandStoreStub{}
	assetSync := &directoryAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 2, TargetID: 8},
		target: &snapshotdomain.ScanTargetRef{ID: 8, Name: "example.com", Type: "domain"},
	}
	service := NewDirectorySnapshotCommandService(store, lookup, assetSync)

	snapshotCount, assetCount, err := service.SaveAndSync(context.Background(), 2, 8, []DirectorySnapshotItem{
		{URL: "https://example.com/admin"},
		{URL: "https://evil.com/admin"},
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

func TestDirectorySnapshotCommandServiceErrors(t *testing.T) {
	service := NewDirectorySnapshotCommandService(&directorySnapshotCommandStoreStub{}, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}, &directoryAssetSyncStub{})

	_, _, err := service.SaveAndSync(context.Background(), 1, 1, []DirectorySnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}

	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1, TargetID: 9}, target: &snapshotdomain.ScanTargetRef{ID: 9, Name: "example.com", Type: "domain"}}
	service = NewDirectorySnapshotCommandService(&directorySnapshotCommandStoreStub{}, lookup, &directoryAssetSyncStub{})
	_, _, err = service.SaveAndSync(context.Background(), 1, 8, []DirectorySnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotTargetMismatch) {
		t.Fatalf("expected ErrSnapshotTargetMismatch, got %v", err)
	}
}
