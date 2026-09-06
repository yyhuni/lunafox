package application

import (
	"context"
	"math"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

func TestDirectoryResultBatchRejectsOutOfScopeItemsBeforeDuplicateSelection(t *testing.T) {
	store := &directorySnapshotCommandStoreStub{}
	assets := &directoryAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 2, TargetID: 8},
		target: &snapshotdomain.ScanTargetRef{ID: 8, Name: "example.com", Type: "domain"},
	}
	service := NewDirectorySnapshotCommandService(store, lookup, assets)
	status := 200
	length := int64(1)
	duration := int64(2)

	summary, err := service.SaveResultBatchContext(context.Background(), 2, 8, []DirectorySnapshotItem{
		{URL: "https://example.com/admin", Status: &status, ContentLength: &length, ContentType: "text/html", Duration: &duration},
		{URL: "https://example.com/admin", Status: &status, ContentLength: &length, ContentType: "text/html", Duration: &duration},
		{URL: "https://outside.test/admin", Status: &status, ContentLength: &length, ContentType: "text/html", Duration: &duration},
	})
	if err != nil {
		t.Fatalf("save result batch failed: %v", err)
	}
	if summary.ReceivedItems != 3 || summary.DuplicateItems != 1 || summary.ScopeFilteredItems != 1 || summary.SnapshotCount != 1 || summary.AssetCount != 1 {
		t.Fatalf("unexpected per-item scope summary: %+v", summary)
	}
	if store.calls != 1 || assets.calls != 1 || len(store.snapshots) != 1 || len(assets.items) != 1 {
		t.Fatalf("valid items did not reach persistence: snapshotCalls=%d assetCalls=%d snapshots=%+v assets=%+v", store.calls, assets.calls, store.snapshots, assets.items)
	}
}

func TestDirectoryResultBatchKeepsExactURLIdentityAndLastCompleteObservation(t *testing.T) {
	store := &directorySnapshotCommandStoreStub{}
	assets := &directoryAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 2, TargetID: 8},
		target: &snapshotdomain.ScanTargetRef{ID: 8, Name: "example.com", Type: "domain"},
	}
	service := NewDirectorySnapshotCommandService(store, lookup, assets)
	firstStatus, lastStatus := 200, 0
	firstLength, lastLength := int64(math.MaxInt64), int64(0)
	firstDuration, lastDuration := int64(math.MaxInt64), int64(0)
	original := "https://Example.com/%00?x=1#frag"

	summary, err := service.SaveResultBatchContext(context.Background(), 2, 8, []DirectorySnapshotItem{
		{URL: original, Status: &firstStatus, ContentLength: &firstLength, ContentType: "text/html", Duration: &firstDuration},
		{URL: "https://example.com/%00?x=1#frag", Status: &firstStatus, ContentLength: &firstLength, ContentType: "application/json", Duration: &firstDuration},
		{URL: original, Status: &lastStatus, ContentLength: &lastLength, ContentType: "", Duration: &lastDuration},
	})
	if err != nil {
		t.Fatalf("save result batch failed: %v", err)
	}
	if summary.ReceivedItems != 3 || summary.DuplicateItems != 1 || summary.SnapshotCount != 2 || summary.AssetCount != 2 || summary.ScopeFilteredItems != 0 {
		t.Fatalf("unexpected materialization summary: %+v", summary)
	}
	if len(store.snapshots) != 2 || len(assets.items) != 2 {
		t.Fatalf("exact-string URL identities collapsed unexpectedly: snapshots=%+v assets=%+v", store.snapshots, assets.items)
	}
	winner := store.snapshots[1]
	if winner.URL != original || winner.Status == nil || *winner.Status != 0 || winner.ContentLength == nil || *winner.ContentLength != 0 || winner.ContentType != "" || winner.Duration == nil || *winner.Duration != 0 {
		t.Fatalf("last complete observation did not replace every mutable field: %+v", winner)
	}
	assetWinner := assets.items[1]
	if assetWinner.URL != original || assetWinner.Status == nil || *assetWinner.Status != 0 || assetWinner.ContentLength == nil || *assetWinner.ContentLength != 0 || assetWinner.ContentType != "" || assetWinner.Duration == nil || *assetWinner.Duration != 0 {
		t.Fatalf("snapshot and asset winners diverged: %+v", assetWinner)
	}
}

func TestDirectoryResultBatchScopeRulesUseRawAuthorityWithoutChangingOriginal(t *testing.T) {
	tests := []struct {
		name        string
		target      snapshotdomain.ScanTargetRef
		rawURL      string
		wantScope   bool
		wantInvalid bool
	}{
		{name: "domain root", target: snapshotdomain.ScanTargetRef{Name: "example.com", Type: "domain"}, rawURL: "https://EXAMPLE.com/admin"},
		{name: "domain subdomain", target: snapshotdomain.ScanTargetRef{Name: "example.com", Type: "domain"}, rawURL: "https://api.example.com/%00"},
		{name: "domain lookalike", target: snapshotdomain.ScanTargetRef{Name: "example.com", Type: "domain"}, rawURL: "https://example.com.attacker.test/admin", wantScope: true},
		{name: "exact IPv4", target: snapshotdomain.ScanTargetRef{Name: "192.0.2.10", Type: "ip"}, rawURL: "http://192.0.2.10:8080/admin"},
		{name: "other IPv4", target: snapshotdomain.ScanTargetRef{Name: "192.0.2.10", Type: "ip"}, rawURL: "http://192.0.2.11/admin", wantScope: true},
		{name: "CIDR network", target: snapshotdomain.ScanTargetRef{Name: "192.0.2.0/24", Type: "cidr"}, rawURL: "http://192.0.2.0/admin"},
		{name: "CIDR broadcast", target: snapshotdomain.ScanTargetRef{Name: "192.0.2.0/24", Type: "cidr"}, rawURL: "http://192.0.2.255/admin"},
		{name: "CIDR outside", target: snapshotdomain.ScanTargetRef{Name: "192.0.2.0/24", Type: "cidr"}, rawURL: "http://192.0.3.1/admin", wantScope: true},
		{name: "CIDR rejects IPv6", target: snapshotdomain.ScanTargetRef{Name: "192.0.2.0/24", Type: "cidr"}, rawURL: "http://[2001:db8::1]/admin", wantInvalid: true},
		{name: "missing hostname", target: snapshotdomain.ScanTargetRef{Name: "example.com", Type: "domain"}, rawURL: "example.com/admin", wantInvalid: true},
		{name: "malformed percent remains literal", target: snapshotdomain.ScanTargetRef{Name: "example.com", Type: "domain"}, rawURL: "https://example.com/%ZZ"},
		{name: "surrounding whitespace is not repaired", target: snapshotdomain.ScanTargetRef{Name: "example.com", Type: "domain"}, rawURL: " https://example.com/admin ", wantInvalid: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			store := &directorySnapshotCommandStoreStub{}
			assets := &directoryAssetSyncStub{}
			lookup := &snapshotScanLookupStub{
				scan:   &snapshotdomain.ScanRef{ID: 2, TargetID: 8},
				target: &testCase.target,
			}
			service := NewDirectorySnapshotCommandService(store, lookup, assets)
			status := 200
			length, duration := int64(1), int64(2)
			summary, err := service.SaveResultBatchContext(context.Background(), 2, 8, []DirectorySnapshotItem{{
				URL: testCase.rawURL, Status: &status, ContentLength: &length, ContentType: "text/html", Duration: &duration,
			}})
			if err != nil {
				t.Fatalf("save result batch failed: %v", err)
			}
			if testCase.wantInvalid || testCase.wantScope {
				if summary.InvalidItems != boolToInt(testCase.wantInvalid) || summary.ScopeFilteredItems != boolToInt(testCase.wantScope) || store.calls != 0 || assets.calls != 0 {
					t.Fatalf("rejection summary=%+v writes=%d/%d", summary, store.calls, assets.calls)
				}
				return
			}
			if len(store.snapshots) != 1 || store.snapshots[0].URL != testCase.rawURL || len(assets.items) != 1 || assets.items[0].URL != testCase.rawURL {
				t.Fatalf("accepted URL was changed: raw=%q snapshots=%+v assets=%+v", testCase.rawURL, store.snapshots, assets.items)
			}
		})
	}
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestDirectoryResultBatchPropagatesCallerContextToEveryDatabaseBoundary(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "result-batch")
	store := &directorySnapshotCommandStoreStub{}
	assets := &directoryAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 2, TargetID: 8},
		target: &snapshotdomain.ScanTargetRef{ID: 8, Name: "example.com", Type: "domain"},
	}
	service := NewDirectorySnapshotCommandService(store, lookup, assets)
	status := 200
	length, duration := int64(1), int64(2)

	_, err := service.SaveResultBatchContext(ctx, 2, 8, []DirectorySnapshotItem{{
		URL: "https://example.com/admin", Status: &status, ContentLength: &length, ContentType: "text/html", Duration: &duration,
	}})
	if err != nil {
		t.Fatalf("save result batch failed: %v", err)
	}
	if store.ctx == nil || store.ctx.Value(contextKey{}) != "result-batch" || assets.ctx == nil || assets.ctx.Value(contextKey{}) != "result-batch" {
		t.Fatalf("caller context did not reach writes: snapshot=%v asset=%v", store.ctx, assets.ctx)
	}
}
