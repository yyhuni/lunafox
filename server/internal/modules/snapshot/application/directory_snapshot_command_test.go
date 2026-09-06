package application

import (
	"context"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type directorySnapshotCommandStoreStub struct {
	snapshots []snapshotdomain.DirectorySnapshot
	err       error
	ctx       context.Context
	calls     int
}

func (stub *directorySnapshotCommandStoreStub) BatchCreate(snapshots []snapshotdomain.DirectorySnapshot) (int64, error) {
	return stub.BatchCreateContext(context.Background(), snapshots)
}

func (stub *directorySnapshotCommandStoreStub) BatchCreateContext(ctx context.Context, snapshots []snapshotdomain.DirectorySnapshot) (int64, error) {
	stub.calls++
	stub.ctx = ctx
	if stub.err != nil {
		return 0, stub.err
	}
	stub.snapshots = append([]snapshotdomain.DirectorySnapshot(nil), snapshots...)
	return int64(len(snapshots)), nil
}

type directoryAssetSyncStub struct {
	items []DirectoryAssetUpsertItem
	err   error
	ctx   context.Context
	calls int
}

func (stub *directoryAssetSyncStub) BatchUpsert(targetID int, items []DirectoryAssetUpsertItem) (int64, error) {
	return stub.BatchUpsertContext(context.Background(), targetID, items)
}

func (stub *directoryAssetSyncStub) BatchUpsertContext(ctx context.Context, targetID int, items []DirectoryAssetUpsertItem) (int64, error) {
	stub.calls++
	stub.ctx = ctx
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

	summary, err := service.SaveAndSync(context.Background(), 2, 8, []DirectorySnapshotItem{
		{URL: "https://example.com/admin"},
		{URL: "https://evil.com/admin"},
	})
	if err != nil {
		t.Fatalf("save and sync failed: %v", err)
	}
	if summary.ReceivedItems != 2 || summary.SnapshotCount != 1 || summary.AssetCount != 1 || summary.ScopeFilteredItems != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if len(store.snapshots) != 1 || len(assetSync.items) != 1 {
		t.Fatalf("unexpected stored counts snapshots=%d items=%d", len(store.snapshots), len(assetSync.items))
	}
}

func TestDirectorySnapshotCommandServiceKeepsLastInScopeItem(t *testing.T) {
	store := &directorySnapshotCommandStoreStub{}
	assets := &directoryAssetSyncStub{}
	service := NewDirectorySnapshotCommandService(store, &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 2, TargetID: 8}, target: &snapshotdomain.ScanTargetRef{ID: 8, Name: "example.com", Type: "domain"}}, assets)
	first, last := 200, 404
	summary, err := service.SaveAndSync(context.Background(), 2, 8, []DirectorySnapshotItem{{URL: "https://example.com/admin", Status: &first}, {URL: "https://example.com/admin", Status: &last}})
	if err != nil {
		t.Fatalf("save and sync failed: %v", err)
	}
	if summary.DuplicateItems != 1 || len(store.snapshots) != 1 || *store.snapshots[0].Status != last || len(assets.items) != 1 || *assets.items[0].Status != last {
		t.Fatalf("unexpected latest winner: summary=%+v snapshots=%+v assets=%+v", summary, store.snapshots, assets.items)
	}
}

func TestDirectorySnapshotCommandServiceErrors(t *testing.T) {
	service := NewDirectorySnapshotCommandService(&directorySnapshotCommandStoreStub{}, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}, &directoryAssetSyncStub{})

	_, err := service.SaveAndSync(context.Background(), 1, 1, []DirectorySnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}

	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1, TargetID: 9}, target: &snapshotdomain.ScanTargetRef{ID: 9, Name: "example.com", Type: "domain"}}
	service = NewDirectorySnapshotCommandService(&directorySnapshotCommandStoreStub{}, lookup, &directoryAssetSyncStub{})
	_, err = service.SaveAndSync(context.Background(), 1, 8, []DirectorySnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotTargetMismatch) {
		t.Fatalf("expected ErrSnapshotTargetMismatch, got %v", err)
	}
}
