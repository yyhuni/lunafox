package application

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type HostPortSnapshotQueryService struct {
	store      HostPortSnapshotQueryStore
	scanLookup SnapshotScanRefLookup
}

func NewHostPortSnapshotQueryService(store HostPortSnapshotQueryStore, scanLookup SnapshotScanRefLookup) *HostPortSnapshotQueryService {
	return &HostPortSnapshotQueryService{store: store, scanLookup: scanLookup}
}

func (service *HostPortSnapshotQueryService) ListByScan(ctx context.Context, scanID, page, pageSize int, filter string) ([]snapshotdomain.HostPortSnapshot, int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrSnapshotScanNotFound
		}
		return nil, 0, err
	}
	return service.store.FindByScanID(scanID, page, pageSize, filter)
}

func (service *HostPortSnapshotQueryService) StreamByScan(ctx context.Context, scanID int) (*sql.Rows, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}
	return service.store.StreamByScanID(scanID)
}

func (service *HostPortSnapshotQueryService) CountByScan(ctx context.Context, scanID int) (int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrSnapshotScanNotFound
		}
		return 0, err
	}
	return service.store.CountByScanID(scanID)
}

func (service *HostPortSnapshotQueryService) ScanRow(ctx context.Context, rows *sql.Rows) (*snapshotdomain.HostPortSnapshot, error) {
	_ = ctx
	return service.store.ScanRow(rows)
}

type HostPortSnapshotCommandService struct {
	store      HostPortSnapshotCommandStore
	scanLookup SnapshotScanRefLookup
	assetSync  HostPortAssetSync
}

func NewHostPortSnapshotCommandService(store HostPortSnapshotCommandStore, scanLookup SnapshotScanRefLookup, assetSync HostPortAssetSync) *HostPortSnapshotCommandService {
	return &HostPortSnapshotCommandService{store: store, scanLookup: scanLookup, assetSync: assetSync}
}

func (service *HostPortSnapshotCommandService) SaveAndSync(ctx context.Context, scanID int, targetID int, items []HostPortSnapshotItem) (snapshotCount int64, assetCount int64, err error) {
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

	snapshots := make([]snapshotdomain.HostPortSnapshot, 0, len(items))
	validItems := make([]HostPortAssetItem, 0, len(items))
	for _, item := range items {
		if snapshotdomain.IsHostPortMatchTarget(item.Host, item.IP, *target) {
			snapshots = append(snapshots, snapshotdomain.HostPortSnapshot{ScanID: scanID, Host: item.Host, IP: item.IP, Port: item.Port})
			validItems = append(validItems, HostPortAssetItem(item))
		}
	}

	if len(snapshots) == 0 {
		return 0, 0, nil
	}
	snapshotCount, err = service.store.BulkCreate(snapshots)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to bulk create snapshots: %w", err)
	}
	assetCount, err = service.assetSync.BulkUpsert(targetID, validItems)
	if err != nil {
		return snapshotCount, 0, fmt.Errorf("failed to sync to asset table: %w", err)
	}
	return snapshotCount, assetCount, nil
}
