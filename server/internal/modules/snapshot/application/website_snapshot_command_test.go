package application

import (
	"context"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type websiteSnapshotCommandStoreStub struct {
	snapshots []snapshotdomain.WebsiteSnapshot
	createErr error
}

func (stub *websiteSnapshotCommandStoreStub) BatchCreateContext(_ context.Context, snapshots []snapshotdomain.WebsiteSnapshot) (int64, error) {
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

func (stub *websiteAssetSyncStub) BatchUpsertContext(_ context.Context, targetID int, items []WebsiteAssetUpsertItem) (int64, error) {
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

	summary, err := service.SaveAndSync(context.Background(), 11, 7, []WebsiteSnapshotItem{
		{URL: "https://example.com/a", Host: "example.com"},
		{URL: "https://evil.com/a", Host: "evil.com"},
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
	if store.snapshots[0].Host != "example.com" {
		t.Fatalf("expected derived host example.com, got %s", store.snapshots[0].Host)
	}
}

func TestWebsiteSnapshotCommandServiceKeepsConfirmedTargetBaselineURLs(t *testing.T) {
	store := &websiteSnapshotCommandStoreStub{}
	assetSync := &websiteAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 11, TargetID: 7},
		target: &snapshotdomain.ScanTargetRef{ID: 7, Name: "example.com", Type: "domain"},
	}
	service := NewWebsiteSnapshotCommandService(store, lookup, assetSync)

	summary, err := service.SaveAndSync(context.Background(), 11, 7, []WebsiteSnapshotItem{
		{URL: "http://example.com", Host: "example.com"},
		{URL: "https://example.com", Host: "example.com"},
	})
	if err != nil {
		t.Fatalf("save and sync failed: %v", err)
	}
	if summary.ScopeFilteredItems != 0 || summary.SnapshotCount != 2 || summary.AssetCount != 2 {
		t.Fatalf("confirmed Target baseline URLs must remain facts: summary=%+v", summary)
	}
	if len(store.snapshots) != 2 || store.snapshots[0].URL != "http://example.com" || store.snapshots[1].URL != "https://example.com" {
		t.Fatalf("unexpected retained baseline URLs: %+v", store.snapshots)
	}
}

func TestWebsiteSnapshotCommandServiceSaveAndSyncCountsScopeFilteredUniqueItems(t *testing.T) {
	store := &websiteSnapshotCommandStoreStub{}
	assetSync := &websiteAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 11, TargetID: 7},
		target: &snapshotdomain.ScanTargetRef{ID: 7, Name: "example.com", Type: "domain"},
	}
	service := NewWebsiteSnapshotCommandService(store, lookup, assetSync)

	summary, err := service.SaveAndSync(context.Background(), 11, 7, []WebsiteSnapshotItem{
		{URL: "https://api.example.com", Host: "api.example.com"},
		{URL: "https://evil.com", Host: "evil.com"},
		{URL: "https://evil.com", Host: "evil.com"},
	})
	if err != nil {
		t.Fatalf("save and sync with summary failed: %v", err)
	}
	if summary.ReceivedItems != 3 || summary.SnapshotCount != 1 || summary.AssetCount != 1 || summary.ScopeFilteredItems != 2 || summary.UnsupportedItems != 0 {
		t.Fatalf("unexpected materialization summary: %+v", summary)
	}
}

func TestWebsiteSnapshotCommandServiceKeepsExactRawURLIdentities(t *testing.T) {
	store := &websiteSnapshotCommandStoreStub{}
	assetSync := &websiteAssetSyncStub{}
	service := NewWebsiteSnapshotCommandService(store, &snapshotScanLookupStub{
		scan: &snapshotdomain.ScanRef{ID: 11, TargetID: 7}, target: &snapshotdomain.ScanTargetRef{ID: 7, Name: "example.com", Type: "domain"},
	}, assetSync)

	summary, err := service.SaveAndSync(context.Background(), 11, 7, []WebsiteSnapshotItem{
		{URL: "https://example.com:443/app", Host: "example.com", Title: "earlier", Tech: []string{"old"}},
		{URL: "https://example.com/app", Host: "example.com", Title: "later"},
		{URL: "https://example.com/app", Host: "wrong.example.com", Title: "invalid"},
	})
	if err != nil {
		t.Fatalf("save and sync failed: %v", err)
	}
	if summary.DuplicateItems != 0 || summary.InvalidItems != 1 || len(store.snapshots) != 2 || len(assetSync.items) != 2 {
		t.Fatalf("unexpected summary or write sets: summary=%+v snapshots=%+v assets=%+v", summary, store.snapshots, assetSync.items)
	}
	if store.snapshots[0].Title != "earlier" || assetSync.items[0].Title != "earlier" || len(store.snapshots[0].Tech) != 1 || len(assetSync.items[0].Tech) != 1 {
		t.Fatalf("explicit default ports must remain separate raw identities: snapshots=%+v assets=%+v", store.snapshots, assetSync.items)
	}
	if store.snapshots[1].Title != "later" || assetSync.items[1].Title != "later" || len(store.snapshots[1].Tech) != 0 || len(assetSync.items[1].Tech) != 0 {
		t.Fatalf("snapshot and asset must share the later exact-key winner: snapshots=%+v assets=%+v", store.snapshots, assetSync.items)
	}
}

func TestWebsiteSnapshotCommandServiceRejectsUnreliableAuthorityWithoutRepair(t *testing.T) {
	store := &websiteSnapshotCommandStoreStub{}
	assets := &websiteAssetSyncStub{}
	service := NewWebsiteSnapshotCommandService(store, &snapshotScanLookupStub{
		scan: &snapshotdomain.ScanRef{ID: 11, TargetID: 7}, target: &snapshotdomain.ScanTargetRef{ID: 7, Name: "example.com", Type: "domain"},
	}, assets)

	summary, err := service.SaveAndSync(context.Background(), 11, 7, []WebsiteSnapshotItem{
		{URL: "https://example.com:443:444/payload", Host: "example.com"},
		{URL: "https://example.com/%00?x=%zz#fragment", Host: "example.com", Title: "raw"},
	})
	if err != nil {
		t.Fatalf("save and sync failed: %v", err)
	}
	if summary.InvalidItems != 1 || summary.SnapshotCount != 1 || summary.AssetCount != 1 {
		t.Fatalf("unexpected authority rejection summary: %+v", summary)
	}
	if len(store.snapshots) != 1 || store.snapshots[0].URL != "https://example.com/%00?x=%zz#fragment" || len(assets.items) != 1 || assets.items[0].URL != store.snapshots[0].URL {
		t.Fatalf("valid raw URL was not preserved: snapshots=%+v assets=%+v", store.snapshots, assets.items)
	}
}

func TestWebsiteSnapshotCommandServiceErrors(t *testing.T) {
	service := NewWebsiteSnapshotCommandService(&websiteSnapshotCommandStoreStub{}, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}, &websiteAssetSyncStub{})

	_, err := service.SaveAndSync(context.Background(), 1, 1, []WebsiteSnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}

	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1, TargetID: 9}, target: &snapshotdomain.ScanTargetRef{ID: 9, Name: "example.com", Type: "domain"}}
	service = NewWebsiteSnapshotCommandService(&websiteSnapshotCommandStoreStub{}, lookup, &websiteAssetSyncStub{})
	_, err = service.SaveAndSync(context.Background(), 1, 8, []WebsiteSnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotTargetMismatch) {
		t.Fatalf("expected ErrSnapshotTargetMismatch, got %v", err)
	}
}
