package application

import (
	"context"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type subdomainSnapshotCommandStoreStub struct {
	snapshots []snapshotdomain.SubdomainSnapshot
	err       error
}

func (stub *subdomainSnapshotCommandStoreStub) BatchCreateContext(_ context.Context, snapshots []snapshotdomain.SubdomainSnapshot) (int64, error) {
	if stub.err != nil {
		return 0, stub.err
	}
	stub.snapshots = append([]snapshotdomain.SubdomainSnapshot(nil), snapshots...)
	return int64(len(snapshots)), nil
}

type subdomainAssetSyncStub struct {
	dnsNames []string
	err      error
}

func (stub *subdomainAssetSyncStub) BatchCreateContext(_ context.Context, targetID int, dnsNames []string) (int, error) {
	_ = targetID
	if stub.err != nil {
		return 0, stub.err
	}
	stub.dnsNames = append([]string(nil), dnsNames...)
	return len(dnsNames), nil
}

func TestSubdomainSnapshotCommandServiceSaveAndSync(t *testing.T) {
	store := &subdomainSnapshotCommandStoreStub{}
	assetSync := &subdomainAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 4, TargetID: 6},
		target: &snapshotdomain.ScanTargetRef{ID: 6, Name: "example.com", Type: "domain"},
	}
	service := NewSubdomainSnapshotCommandService(store, lookup, assetSync)

	summary, err := service.SaveAndSync(context.Background(), 4, 6, []SubdomainSnapshotItem{
		{DNSName: "api.example.com"},
		{DNSName: "evil.com"},
	})
	if err != nil {
		t.Fatalf("save and sync failed: %v", err)
	}
	if summary.ReceivedItems != 2 || summary.SnapshotCount != 1 || summary.AssetCount != 1 || summary.ScopeFilteredItems != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if len(store.snapshots) != 1 || len(assetSync.dnsNames) != 1 {
		t.Fatalf("unexpected stored counts snapshots=%d dnsNames=%d", len(store.snapshots), len(assetSync.dnsNames))
	}
}

func TestSubdomainSnapshotCommandServiceRejectsNonCanonicalWithoutRepair(t *testing.T) {
	store := &subdomainSnapshotCommandStoreStub{}
	assetSync := &subdomainAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 4, TargetID: 6},
		target: &snapshotdomain.ScanTargetRef{ID: 6, Name: "example.com", Type: "domain"},
	}
	service := NewSubdomainSnapshotCommandService(store, lookup, assetSync)

	summary, err := service.SaveAndSync(context.Background(), 4, 6, []SubdomainSnapshotItem{
		{DNSName: "Api.Example.COM."},
		{DNSName: "api.example.com"},
		{DNSName: "127.0.0.1"},
	})
	if err != nil {
		t.Fatalf("save and sync failed: %v", err)
	}
	if summary.ReceivedItems != 3 || summary.SnapshotCount != 1 || summary.AssetCount != 1 || summary.InvalidItems != 2 || summary.DuplicateItems != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if len(store.snapshots) != 1 || store.snapshots[0].DNSName != "api.example.com" {
		t.Fatalf("unexpected snapshots: %+v", store.snapshots)
	}
	if len(assetSync.dnsNames) != 1 || assetSync.dnsNames[0] != "api.example.com" {
		t.Fatalf("unexpected asset sync names: %+v", assetSync.dnsNames)
	}
}

func TestSubdomainSnapshotCommandServiceSaveAndSyncCountsScopeFilteredUniqueItems(t *testing.T) {
	store := &subdomainSnapshotCommandStoreStub{}
	assetSync := &subdomainAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 4, TargetID: 6},
		target: &snapshotdomain.ScanTargetRef{ID: 6, Name: "example.com", Type: "domain"},
	}
	service := NewSubdomainSnapshotCommandService(store, lookup, assetSync)

	summary, err := service.SaveAndSync(context.Background(), 4, 6, []SubdomainSnapshotItem{
		{DNSName: "api.example.com"},
		{DNSName: "evil.com"},
		{DNSName: "evil.com"},
	})
	if err != nil {
		t.Fatalf("save and sync with summary failed: %v", err)
	}
	if summary.ReceivedItems != 3 || summary.SnapshotCount != 1 || summary.AssetCount != 1 || summary.ScopeFilteredItems != 2 || summary.DuplicateItems != 0 || summary.InvalidItems != 0 || summary.UnsupportedItems != 0 {
		t.Fatalf("unexpected materialization summary: %+v", summary)
	}
}

func TestSubdomainSnapshotCommandServiceExcludesTargetRootFromFinalizedFacts(t *testing.T) {
	store := &subdomainSnapshotCommandStoreStub{}
	assetSync := &subdomainAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 4, TargetID: 6},
		target: &snapshotdomain.ScanTargetRef{ID: 6, Name: "example.com", Type: "domain"},
	}
	service := NewSubdomainSnapshotCommandService(store, lookup, assetSync)

	summary, err := service.SaveAndSync(context.Background(), 4, 6, []SubdomainSnapshotItem{
		{DNSName: "example.com"},
		{DNSName: "api.example.com"},
	})
	if err != nil {
		t.Fatalf("save and sync failed: %v", err)
	}
	if summary.ScopeFilteredItems != 1 || len(store.snapshots) != 1 || store.snapshots[0].DNSName != "api.example.com" {
		t.Fatalf("target root must not be finalized as a subdomain fact: summary=%+v snapshots=%+v", summary, store.snapshots)
	}
}

func TestSubdomainSnapshotCommandServiceErrors(t *testing.T) {
	service := NewSubdomainSnapshotCommandService(&subdomainSnapshotCommandStoreStub{}, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}, &subdomainAssetSyncStub{})

	_, err := service.SaveAndSync(context.Background(), 1, 1, []SubdomainSnapshotItem{{DNSName: "api.example.com"}})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}

	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1, TargetID: 9}, target: &snapshotdomain.ScanTargetRef{ID: 9, Name: "example.com", Type: "domain"}}
	service = NewSubdomainSnapshotCommandService(&subdomainSnapshotCommandStoreStub{}, lookup, &subdomainAssetSyncStub{})
	_, err = service.SaveAndSync(context.Background(), 1, 8, []SubdomainSnapshotItem{{DNSName: "api.example.com"}})
	if !errors.Is(err, ErrSnapshotTargetMismatch) {
		t.Fatalf("expected ErrSnapshotTargetMismatch, got %v", err)
	}

	lookup.target.Type = "ip"
	_, err = service.SaveAndSync(context.Background(), 1, 9, []SubdomainSnapshotItem{{DNSName: "api.example.com"}})
	if !errors.Is(err, ErrSubdomainSnapshotInvalidTargetType) {
		t.Fatalf("expected ErrSubdomainSnapshotInvalidTargetType, got %v", err)
	}
}
