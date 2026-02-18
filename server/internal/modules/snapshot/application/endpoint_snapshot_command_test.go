package application

import (
	"context"
	"errors"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
	"testing"
)

type endpointSnapshotCommandStoreStub struct {
	snapshots []snapshotdomain.EndpointSnapshot
	err       error
}

func (stub *endpointSnapshotCommandStoreStub) BulkCreate(snapshots []snapshotdomain.EndpointSnapshot) (int64, error) {
	if stub.err != nil {
		return 0, stub.err
	}
	stub.snapshots = append([]snapshotdomain.EndpointSnapshot(nil), snapshots...)
	return int64(len(snapshots)), nil
}

type endpointAssetSyncStub struct {
	items []EndpointAssetUpsertItem
	err   error
}

func (stub *endpointAssetSyncStub) BulkUpsert(targetID int, items []EndpointAssetUpsertItem) (int64, error) {
	_ = targetID
	if stub.err != nil {
		return 0, stub.err
	}
	stub.items = append([]EndpointAssetUpsertItem(nil), items...)
	return int64(len(items)), nil
}

func TestEndpointSnapshotCommandServiceSaveAndSync(t *testing.T) {
	store := &endpointSnapshotCommandStoreStub{}
	assetSync := &endpointAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 5, TargetID: 7},
		target: &snapshotdomain.ScanTargetRef{ID: 7, Name: "example.com", Type: "domain"},
	}
	service := NewEndpointSnapshotCommandService(store, lookup, assetSync)

	snapshotCount, assetCount, err := service.SaveAndSync(context.Background(), 5, 7, []EndpointSnapshotItem{
		{URL: "https://example.com/api"},
		{URL: "https://evil.com/api"},
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

func TestEndpointSnapshotCommandServiceErrors(t *testing.T) {
	service := NewEndpointSnapshotCommandService(&endpointSnapshotCommandStoreStub{}, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}, &endpointAssetSyncStub{})

	_, _, err := service.SaveAndSync(context.Background(), 1, 1, []EndpointSnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}

	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1, TargetID: 9}, target: &snapshotdomain.ScanTargetRef{ID: 9, Name: "example.com", Type: "domain"}}
	service = NewEndpointSnapshotCommandService(&endpointSnapshotCommandStoreStub{}, lookup, &endpointAssetSyncStub{})
	_, _, err = service.SaveAndSync(context.Background(), 1, 8, []EndpointSnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotTargetMismatch) {
		t.Fatalf("expected ErrSnapshotTargetMismatch, got %v", err)
	}
}
