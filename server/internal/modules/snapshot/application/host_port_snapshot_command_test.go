package application

import (
	"context"
	"errors"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
	"testing"
)

type hostPortSnapshotCommandStoreStub struct {
	snapshots []snapshotdomain.HostPortSnapshot
	err       error
}

func (stub *hostPortSnapshotCommandStoreStub) BulkCreate(snapshots []snapshotdomain.HostPortSnapshot) (int64, error) {
	if stub.err != nil {
		return 0, stub.err
	}
	stub.snapshots = append([]snapshotdomain.HostPortSnapshot(nil), snapshots...)
	return int64(len(snapshots)), nil
}

type hostPortAssetSyncStub struct {
	items []HostPortAssetItem
	err   error
}

func (stub *hostPortAssetSyncStub) BulkUpsert(targetID int, items []HostPortAssetItem) (int64, error) {
	_ = targetID
	if stub.err != nil {
		return 0, stub.err
	}
	stub.items = append([]HostPortAssetItem(nil), items...)
	return int64(len(items)), nil
}

func TestHostPortSnapshotCommandServiceSaveAndSync(t *testing.T) {
	store := &hostPortSnapshotCommandStoreStub{}
	assetSync := &hostPortAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 5, TargetID: 7},
		target: &snapshotdomain.ScanTargetRef{ID: 7, Name: "example.com", Type: "domain"},
	}
	service := NewHostPortSnapshotCommandService(store, lookup, assetSync)

	snapshotCount, assetCount, err := service.SaveAndSync(context.Background(), 5, 7, []HostPortSnapshotItem{
		{Host: "a.example.com", IP: "1.1.1.1", Port: 443},
		{Host: "evil.com", IP: "2.2.2.2", Port: 80},
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

func TestHostPortSnapshotCommandServiceErrors(t *testing.T) {
	service := NewHostPortSnapshotCommandService(&hostPortSnapshotCommandStoreStub{}, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}, &hostPortAssetSyncStub{})

	_, _, err := service.SaveAndSync(context.Background(), 1, 1, []HostPortSnapshotItem{{Host: "a.example.com", IP: "1.1.1.1", Port: 443}})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}

	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1, TargetID: 9}, target: &snapshotdomain.ScanTargetRef{ID: 9, Name: "example.com", Type: "domain"}}
	service = NewHostPortSnapshotCommandService(&hostPortSnapshotCommandStoreStub{}, lookup, &hostPortAssetSyncStub{})
	_, _, err = service.SaveAndSync(context.Background(), 1, 8, []HostPortSnapshotItem{{Host: "a.example.com", IP: "1.1.1.1", Port: 443}})
	if !errors.Is(err, ErrSnapshotTargetMismatch) {
		t.Fatalf("expected ErrSnapshotTargetMismatch, got %v", err)
	}
}

func TestIsHostPortMatchTarget(t *testing.T) {
	if !snapshotdomain.IsHostPortMatchTarget("a.example.com", "1.1.1.1", snapshotdomain.ScanTargetRef{Name: "example.com", Type: "domain"}) {
		t.Fatalf("expected domain match")
	}
	if !snapshotdomain.IsHostPortMatchTarget("x", "1.1.1.1", snapshotdomain.ScanTargetRef{Name: "1.1.1.1", Type: "ip"}) {
		t.Fatalf("expected ip match")
	}
	if !snapshotdomain.IsHostPortMatchTarget("x", "10.0.0.5", snapshotdomain.ScanTargetRef{Name: "10.0.0.0/24", Type: "cidr"}) {
		t.Fatalf("expected cidr match")
	}
}
