package application

import (
	"context"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type screenshotSnapshotCommandStoreStub struct {
	snapshots []snapshotdomain.ScreenshotSnapshot
	err       error
}

func (stub *screenshotSnapshotCommandStoreStub) BatchUpsertContext(_ context.Context, snapshots []snapshotdomain.ScreenshotSnapshot) (int64, error) {
	if stub.err != nil {
		return 0, stub.err
	}
	stub.snapshots = append([]snapshotdomain.ScreenshotSnapshot(nil), snapshots...)
	return int64(len(snapshots)), nil
}

type screenshotAssetSyncStub struct {
	items []ScreenshotAssetItem
	err   error
}

type screenshotMaterializationCoordinatorStub struct {
	calls int
	err   error
}

func (stub *screenshotMaterializationCoordinatorStub) Materialize(ctx context.Context, _ int, _ int, persist func(context.Context) error) error {
	stub.calls++
	if stub.err != nil {
		return stub.err
	}
	return persist(ctx)
}

type screenshotSnapshotContextStoreStub struct {
	screenshotSnapshotCommandStoreStub
	context context.Context
}

func (stub *screenshotSnapshotContextStoreStub) BatchUpsertContext(ctx context.Context, snapshots []snapshotdomain.ScreenshotSnapshot) (int64, error) {
	stub.context = ctx
	return stub.screenshotSnapshotCommandStoreStub.BatchUpsertContext(ctx, snapshots)
}

type screenshotAssetContextSyncStub struct {
	screenshotAssetSyncStub
	context context.Context
}

func (stub *screenshotAssetContextSyncStub) BatchUpsertContext(ctx context.Context, targetID int, request *ScreenshotAssetUpsertRequest) (int64, error) {
	stub.context = ctx
	return stub.screenshotAssetSyncStub.BatchUpsertContext(ctx, targetID, request)
}

func validScreenshotImage() []byte {
	return []byte{
		'R', 'I', 'F', 'F', 22, 0, 0, 0, 'W', 'E', 'B', 'P',
		'V', 'P', '8', ' ', 10, 0, 0, 0,
		0, 0, 0, 0x9d, 0x01, 0x2a, 1, 0, 1, 0,
	}
}

func (stub *screenshotAssetSyncStub) BatchUpsertContext(_ context.Context, targetID int, req *ScreenshotAssetUpsertRequest) (int64, error) {
	_ = targetID
	if stub.err != nil {
		return 0, stub.err
	}
	stub.items = append([]ScreenshotAssetItem(nil), req.Screenshots...)
	return int64(len(req.Screenshots)), nil
}

func TestScreenshotSnapshotCommandServiceSaveAndSync(t *testing.T) {
	store := &screenshotSnapshotCommandStoreStub{}
	assetSync := &screenshotAssetSyncStub{}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 4, TargetID: 6},
		target: &snapshotdomain.ScanTargetRef{ID: 6, Name: "example.com", Type: "domain"},
	}
	service := NewScreenshotSnapshotCommandService(store, lookup, assetSync)

	summary, err := service.SaveAndSync(context.Background(), 4, 6, []ScreenshotSnapshotItem{
		{URL: "https://example.com/a", Image: validScreenshotImage()},
		{URL: "https://evil.com/a", Image: validScreenshotImage()},
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

func TestScreenshotSnapshotCommandServiceExcludesInvalidLaterImageBeforeSelection(t *testing.T) {
	store := &screenshotSnapshotCommandStoreStub{}
	assets := &screenshotAssetSyncStub{}
	service := NewScreenshotSnapshotCommandService(store, &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 4, TargetID: 6}, target: &snapshotdomain.ScanTargetRef{ID: 6, Name: "example.com", Type: "domain"}}, assets)
	status := int16(200)
	summary, err := service.SaveAndSync(context.Background(), 4, 6, []ScreenshotSnapshotItem{{URL: "https://example.com/a", StatusCode: &status, Image: validScreenshotImage()}, {URL: "https://example.com/a", Image: []byte("not-an-image")}})
	if err != nil {
		t.Fatalf("save and sync failed: %v", err)
	}
	if summary.InvalidItems != 1 || summary.DuplicateItems != 0 || len(store.snapshots) != 1 || string(store.snapshots[0].Image) != string(validScreenshotImage()) || len(assets.items) != 1 || string(assets.items[0].Image) != string(validScreenshotImage()) {
		t.Fatalf("invalid image must not suppress valid winner: summary=%+v snapshots=%+v assets=%+v", summary, store.snapshots, assets.items)
	}
}

func TestScreenshotSnapshotCommandServiceSaveResultBatchUsesOuterTransactionContext(t *testing.T) {
	type contextKey struct{}
	store := &screenshotSnapshotContextStoreStub{}
	assets := &screenshotAssetContextSyncStub{}
	coordinator := &screenshotMaterializationCoordinatorStub{err: errors.New("nested transaction must not start")}
	lookup := &snapshotScanLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 4, TargetID: 6},
		target: &snapshotdomain.ScanTargetRef{ID: 6, Name: "example.com", Type: "domain"},
	}
	service := NewScreenshotSnapshotCommandService(store, lookup, assets, coordinator)
	ctx := context.WithValue(context.Background(), contextKey{}, "result-ingest-transaction")

	summary, err := service.SaveResultBatchContext(ctx, 4, 6, []ScreenshotSnapshotItem{{URL: "https://example.com/a", Image: validScreenshotImage()}})
	if err != nil {
		t.Fatalf("save result batch: %v", err)
	}
	if coordinator.calls != 0 {
		t.Fatalf("result batch started nested coordinator transaction %d times", coordinator.calls)
	}
	if store.context == nil || store.context.Value(contextKey{}) != "result-ingest-transaction" || assets.context == nil || assets.context.Value(contextKey{}) != "result-ingest-transaction" {
		t.Fatalf("result batch did not preserve outer transaction context: store=%v assets=%v", store.context, assets.context)
	}
	if summary.SnapshotCount != 1 || summary.AssetCount != 1 {
		t.Fatalf("unexpected result batch summary: %+v", summary)
	}
}

func TestScreenshotSnapshotCommandServiceErrors(t *testing.T) {
	service := NewScreenshotSnapshotCommandService(&screenshotSnapshotCommandStoreStub{}, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}, &screenshotAssetSyncStub{})

	_, err := service.SaveAndSync(context.Background(), 1, 1, []ScreenshotSnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}

	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1, TargetID: 9}, target: &snapshotdomain.ScanTargetRef{ID: 9, Name: "example.com", Type: "domain"}}
	service = NewScreenshotSnapshotCommandService(&screenshotSnapshotCommandStoreStub{}, lookup, &screenshotAssetSyncStub{})
	_, err = service.SaveAndSync(context.Background(), 1, 8, []ScreenshotSnapshotItem{{URL: "https://example.com"}})
	if !errors.Is(err, ErrSnapshotTargetMismatch) {
		t.Fatalf("expected ErrSnapshotTargetMismatch, got %v", err)
	}
}
