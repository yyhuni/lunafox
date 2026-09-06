package application

import (
	"context"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type endpointSnapshotCommandStoreStub struct {
	snapshots []snapshotdomain.EndpointSnapshot
	err       error
	ctx       context.Context
}

func (stub *endpointSnapshotCommandStoreStub) BatchCreateContext(ctx context.Context, snapshots []snapshotdomain.EndpointSnapshot) (int64, error) {
	stub.ctx = ctx
	if stub.err != nil {
		return 0, stub.err
	}
	stub.snapshots = append([]snapshotdomain.EndpointSnapshot(nil), snapshots...)
	return int64(len(snapshots)), nil
}

type endpointAssetSyncStub struct {
	items []EndpointAssetUpsertItem
	err   error
	ctx   context.Context
}

func (stub *endpointAssetSyncStub) BatchUpsertContext(ctx context.Context, targetID int, items []EndpointAssetUpsertItem) (int64, error) {
	stub.ctx = ctx
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

	summary, err := service.SaveAndSync(context.Background(), 5, 7, []EndpointSnapshotItem{
		{URL: "https://example.com/api", Host: "example.com"},
		{URL: "https://evil.com/api", Host: "evil.com"},
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

func TestEndpointSnapshotCommandServiceKeepsLastInScopeItem(t *testing.T) {
	store := &endpointSnapshotCommandStoreStub{}
	assets := &endpointAssetSyncStub{}
	service := NewEndpointSnapshotCommandService(store, &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 5, TargetID: 7}, target: &snapshotdomain.ScanTargetRef{ID: 7, Name: "example.com", Type: "domain"}}, assets)
	summary, err := service.SaveAndSync(context.Background(), 5, 7, []EndpointSnapshotItem{
		{URL: "https://example.com/api", Host: "example.com", Title: "first", Location: "https://example.com/old", Tech: []string{"old"}, ResponseBody: "old", ResponseBodyTruncated: true, ResponseHeaders: "old", ResponseHeadersTruncated: true},
		{URL: "https://example.com/api", Host: "example.com", Tech: []string{}, ResponseBodyTruncated: false, ResponseHeadersTruncated: false},
	})
	if err != nil {
		t.Fatalf("save and sync failed: %v", err)
	}
	if summary.DuplicateItems != 1 || len(store.snapshots) != 1 || len(assets.items) != 1 {
		t.Fatalf("unexpected latest winner: summary=%+v snapshots=%+v assets=%+v", summary, store.snapshots, assets.items)
	}
	if store.snapshots[0].Title != "" || store.snapshots[0].Location != "" || len(store.snapshots[0].Tech) != 0 || store.snapshots[0].ResponseBody != "" || store.snapshots[0].ResponseBodyTruncated || store.snapshots[0].ResponseHeaders != "" || store.snapshots[0].ResponseHeadersTruncated {
		t.Fatalf("snapshot must fully replace cleared fields: %+v", store.snapshots[0])
	}
	if assets.items[0].Title != "" || assets.items[0].Location != "" || len(assets.items[0].Tech) != 0 || assets.items[0].ResponseBody != "" || assets.items[0].ResponseBodyTruncated || assets.items[0].ResponseHeaders != "" || assets.items[0].ResponseHeadersTruncated {
		t.Fatalf("asset must fully replace cleared fields: %+v", assets.items[0])
	}
}

func TestEndpointSnapshotCommandServiceFiltersScopeBeforeDuplicateFolding(t *testing.T) {
	store := &endpointSnapshotCommandStoreStub{}
	assets := &endpointAssetSyncStub{}
	service := NewEndpointSnapshotCommandService(store, &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 5, TargetID: 7}, target: &snapshotdomain.ScanTargetRef{ID: 7, Name: "example.com", Type: "domain"}}, assets)

	summary, err := service.SaveAndSync(context.Background(), 5, 7, []EndpointSnapshotItem{
		{URL: "https://api.example.com/path", Host: "api.example.com", Title: "first"},
		{URL: "https://evil.example.net/path", Host: "evil.example.net", Title: "outside-first"},
		{URL: "https://evil.example.net/path", Host: "evil.example.net", Title: "outside-last"},
		{URL: "https://api.example.com/path", Host: "api.example.com", Title: "last"},
	})
	if err != nil {
		t.Fatalf("save and sync failed: %v", err)
	}
	if summary.ReceivedItems != 4 || summary.ScopeFilteredItems != 2 || summary.DuplicateItems != 1 || summary.SnapshotCount != 1 || summary.AssetCount != 1 {
		t.Fatalf("scope filtering must precede duplicate folding: %+v", summary)
	}
	if len(store.snapshots) != 1 || store.snapshots[0].Title != "last" || len(assets.items) != 1 || assets.items[0].Title != "last" {
		t.Fatalf("expected only the final in-scope record to materialize: snapshots=%+v assets=%+v", store.snapshots, assets.items)
	}
}

func TestEndpointSnapshotCommandServiceRejectsUnreliableAuthorityBeforeWinnerSelection(t *testing.T) {
	store := &endpointSnapshotCommandStoreStub{}
	assets := &endpointAssetSyncStub{}
	service := NewEndpointSnapshotCommandService(store, &snapshotScanLookupStub{
		scan: &snapshotdomain.ScanRef{ID: 5, TargetID: 7}, target: &snapshotdomain.ScanTargetRef{ID: 7, Name: "example.com", Type: "domain"},
	}, assets)

	summary, err := service.SaveAndSync(context.Background(), 5, 7, []EndpointSnapshotItem{
		{URL: "https://example.com/api", Host: "example.com", Title: "valid"},
		{URL: "https://example.com:443:444/api", Host: "example.com", Title: "ambiguous"},
		{URL: "https://example.com/api", Host: "other.example.com", Title: "invalid-later"},
	})
	if err != nil {
		t.Fatalf("save and sync failed: %v", err)
	}
	if summary.InvalidItems != 2 || summary.DuplicateItems != 0 || summary.SnapshotCount != 1 || summary.AssetCount != 1 {
		t.Fatalf("invalid later item must not replace valid winner: %+v", summary)
	}
	if len(store.snapshots) != 1 || store.snapshots[0].Title != "valid" || len(assets.items) != 1 || assets.items[0].Title != "valid" {
		t.Fatalf("valid winner changed after invalid records: snapshots=%+v assets=%+v", store.snapshots, assets.items)
	}
}

func TestEndpointSnapshotCommandServiceErrors(t *testing.T) {
	service := NewEndpointSnapshotCommandService(&endpointSnapshotCommandStoreStub{}, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}, &endpointAssetSyncStub{})

	_, err := service.SaveAndSync(context.Background(), 1, 1, []EndpointSnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}

	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1, TargetID: 9}, target: &snapshotdomain.ScanTargetRef{ID: 9, Name: "example.com", Type: "domain"}}
	service = NewEndpointSnapshotCommandService(&endpointSnapshotCommandStoreStub{}, lookup, &endpointAssetSyncStub{})
	_, err = service.SaveAndSync(context.Background(), 1, 8, []EndpointSnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotTargetMismatch) {
		t.Fatalf("expected ErrSnapshotTargetMismatch, got %v", err)
	}
}
