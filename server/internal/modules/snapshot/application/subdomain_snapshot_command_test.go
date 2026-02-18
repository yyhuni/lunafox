package application

import (
	"context"
	"errors"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
	"testing"
)

type subdomainSnapshotCommandStoreStub struct {
	snapshots []snapshotdomain.SubdomainSnapshot
	err       error
}

func (stub *subdomainSnapshotCommandStoreStub) BulkCreate(snapshots []snapshotdomain.SubdomainSnapshot) (int64, error) {
	if stub.err != nil {
		return 0, stub.err
	}
	stub.snapshots = append([]snapshotdomain.SubdomainSnapshot(nil), snapshots...)
	return int64(len(snapshots)), nil
}

type subdomainAssetSyncStub struct {
	names []string
	err   error
}

func (stub *subdomainAssetSyncStub) BulkCreate(targetID int, names []string) (int, error) {
	_ = targetID
	if stub.err != nil {
		return 0, stub.err
	}
	stub.names = append([]string(nil), names...)
	return len(names), nil
}

func TestSubdomainSnapshotCommandServiceSaveAndSync(t *testing.T) {
	store := &subdomainSnapshotCommandStoreStub{}
	assetSync := &subdomainAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 4, TargetID: 6},
		target: &snapshotdomain.ScanTargetRef{ID: 6, Name: "example.com", Type: "domain"},
	}
	service := NewSubdomainSnapshotCommandService(store, lookup, assetSync)

	snapshotCount, assetCount, err := service.SaveAndSync(context.Background(), 4, 6, []SubdomainSnapshotItem{
		{Name: "api.example.com"},
		{Name: "evil.com"},
	})
	if err != nil {
		t.Fatalf("save and sync failed: %v", err)
	}
	if snapshotCount != 1 || assetCount != 1 {
		t.Fatalf("unexpected counts snapshot=%d asset=%d", snapshotCount, assetCount)
	}
	if len(store.snapshots) != 1 || len(assetSync.names) != 1 {
		t.Fatalf("unexpected stored counts snapshots=%d names=%d", len(store.snapshots), len(assetSync.names))
	}
}

func TestSubdomainSnapshotCommandServiceErrors(t *testing.T) {
	service := NewSubdomainSnapshotCommandService(&subdomainSnapshotCommandStoreStub{}, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}, &subdomainAssetSyncStub{})

	_, _, err := service.SaveAndSync(context.Background(), 1, 1, []SubdomainSnapshotItem{{Name: "api.example.com"}})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}

	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1, TargetID: 9}, target: &snapshotdomain.ScanTargetRef{ID: 9, Name: "example.com", Type: "domain"}}
	service = NewSubdomainSnapshotCommandService(&subdomainSnapshotCommandStoreStub{}, lookup, &subdomainAssetSyncStub{})
	_, _, err = service.SaveAndSync(context.Background(), 1, 8, []SubdomainSnapshotItem{{Name: "api.example.com"}})
	if !errors.Is(err, ErrSnapshotTargetMismatch) {
		t.Fatalf("expected ErrSnapshotTargetMismatch, got %v", err)
	}

	lookup.target.Type = "ip"
	_, _, err = service.SaveAndSync(context.Background(), 1, 9, []SubdomainSnapshotItem{{Name: "api.example.com"}})
	if !errors.Is(err, ErrSubdomainSnapshotInvalidTargetType) {
		t.Fatalf("expected ErrSubdomainSnapshotInvalidTargetType, got %v", err)
	}
}
