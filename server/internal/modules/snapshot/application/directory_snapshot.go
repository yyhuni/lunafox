package application

import (
	"context"
	"database/sql"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type DirectorySnapshotQueryService struct {
	store      DirectorySnapshotQueryStore
	scanLookup SnapshotScanRefLookup
}

func NewDirectorySnapshotQueryService(store DirectorySnapshotQueryStore, scanLookup SnapshotScanRefLookup) *DirectorySnapshotQueryService {
	return &DirectorySnapshotQueryService{store: store, scanLookup: scanLookup}
}

func (service *DirectorySnapshotQueryService) ListByScan(ctx context.Context, scanID, page, pageSize int, filter string) ([]snapshotdomain.DirectorySnapshot, int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrSnapshotScanNotFound
		}
		return nil, 0, err
	}
	return service.store.FindByScanID(scanID, page, pageSize, filter)
}

func (service *DirectorySnapshotQueryService) StreamByScan(ctx context.Context, scanID int) (*sql.Rows, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}
	return service.store.StreamByScanID(scanID)
}

func (service *DirectorySnapshotQueryService) CountByScan(ctx context.Context, scanID int) (int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrSnapshotScanNotFound
		}
		return 0, err
	}
	return service.store.CountByScanID(scanID)
}

func (service *DirectorySnapshotQueryService) ScanRow(ctx context.Context, rows *sql.Rows) (*snapshotdomain.DirectorySnapshot, error) {
	_ = ctx
	return service.store.ScanRow(rows)
}

type DirectorySnapshotCommandService struct {
	store      DirectorySnapshotCommandStore
	scanLookup SnapshotScanRefLookup
	assetSync  DirectoryAssetSync
}

func NewDirectorySnapshotCommandService(store DirectorySnapshotCommandStore, scanLookup SnapshotScanRefLookup, assetSync DirectoryAssetSync) *DirectorySnapshotCommandService {
	return &DirectorySnapshotCommandService{store: store, scanLookup: scanLookup, assetSync: assetSync}
}

func (service *DirectorySnapshotCommandService) SaveAndSync(ctx context.Context, scanID int, targetID int, items []DirectorySnapshotItem) (snapshotCount int64, assetCount int64, err error) {
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

	snapshots := make([]snapshotdomain.DirectorySnapshot, 0, len(items))
	validItems := make([]DirectoryAssetUpsertItem, 0, len(items))
	for _, item := range items {
		if !snapshotdomain.IsURLMatchTarget(item.URL, *target) {
			continue
		}
		snapshots = append(snapshots, snapshotdomain.DirectorySnapshot{ScanID: scanID, URL: item.URL, Status: item.Status, ContentLength: item.ContentLength, ContentType: item.ContentType, Duration: item.Duration})
		validItems = append(validItems, DirectoryAssetUpsertItem{URL: item.URL, Status: item.Status, ContentLength: item.ContentLength, ContentType: item.ContentType, Duration: item.Duration})
	}

	if len(snapshots) == 0 {
		return 0, 0, nil
	}
	snapshotCount, err = service.store.BulkCreate(snapshots)
	if err != nil {
		return 0, 0, err
	}
	assetCount, err = service.assetSync.BulkUpsert(targetID, validItems)
	if err != nil {
		return snapshotCount, 0, nil
	}
	return snapshotCount, assetCount, nil
}
