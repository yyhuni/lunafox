package application

import (
	"context"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type ScreenshotSnapshotQueryService struct {
	store      ScreenshotSnapshotQueryStore
	scanLookup SnapshotScanRefLookup
}

func NewScreenshotSnapshotQueryService(store ScreenshotSnapshotQueryStore, scanLookup SnapshotScanRefLookup) *ScreenshotSnapshotQueryService {
	return &ScreenshotSnapshotQueryService{store: store, scanLookup: scanLookup}
}

func (service *ScreenshotSnapshotQueryService) ListByScan(ctx context.Context, scanID, page, pageSize int, filter string) ([]snapshotdomain.ScreenshotSnapshot, int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrSnapshotScanNotFound
		}
		return nil, 0, err
	}
	return service.store.FindByScanID(scanID, page, pageSize, filter)
}

func (service *ScreenshotSnapshotQueryService) GetByID(ctx context.Context, scanID int, id int) (*snapshotdomain.ScreenshotSnapshot, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}
	item, err := service.store.FindByIDAndScanID(id, scanID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrScreenshotSnapshotNotFound
		}
		return nil, err
	}
	return item, nil
}

type ScreenshotSnapshotCommandService struct {
	store      ScreenshotSnapshotCommandStore
	scanLookup SnapshotScanRefLookup
	assetSync  ScreenshotAssetSync
}

func NewScreenshotSnapshotCommandService(store ScreenshotSnapshotCommandStore, scanLookup SnapshotScanRefLookup, assetSync ScreenshotAssetSync) *ScreenshotSnapshotCommandService {
	return &ScreenshotSnapshotCommandService{store: store, scanLookup: scanLookup, assetSync: assetSync}
}

func (service *ScreenshotSnapshotCommandService) SaveAndSync(ctx context.Context, scanID int, targetID int, items []ScreenshotSnapshotItem) (snapshotCount int64, assetCount int64, err error) {
	_ = ctx
	if len(items) == 0 {
		return 0, 0, nil
	}

	scan, err := service.scanLookup.GetScanRefByID(scanID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, 0, ErrSnapshotScanNotFound
		}
		return 0, 0, err
	}
	if scan.TargetID != targetID {
		return 0, 0, ErrSnapshotTargetMismatch
	}

	target, err := service.scanLookup.GetTargetRefByScanID(scanID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, 0, ErrSnapshotScanNotFound
		}
		return 0, 0, err
	}

	snapshots := make([]snapshotdomain.ScreenshotSnapshot, 0, len(items))
	assetItems := make([]ScreenshotAssetItem, 0, len(items))
	for _, item := range items {
		if !snapshotdomain.IsURLMatchTarget(item.URL, *target) {
			continue
		}
		snapshots = append(snapshots, snapshotdomain.ScreenshotSnapshot{ScanID: scanID, URL: item.URL, StatusCode: item.StatusCode, Image: item.Image})
		assetItems = append(assetItems, ScreenshotAssetItem(item))
	}

	if len(snapshots) == 0 {
		return 0, 0, nil
	}
	snapshotCount, err = service.store.BulkUpsert(snapshots)
	if err != nil {
		return 0, 0, err
	}
	assetCount, err = service.assetSync.BulkUpsert(targetID, &ScreenshotAssetUpsertRequest{Screenshots: assetItems})
	if err != nil {
		return snapshotCount, 0, nil
	}
	return snapshotCount, assetCount, nil
}
