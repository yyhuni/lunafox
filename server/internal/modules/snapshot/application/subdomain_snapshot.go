package application

import (
	"context"
	"database/sql"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type SubdomainSnapshotQueryService struct {
	store      SubdomainSnapshotQueryStore
	scanLookup SnapshotScanRefLookup
}

func NewSubdomainSnapshotQueryService(store SubdomainSnapshotQueryStore, scanLookup SnapshotScanRefLookup) *SubdomainSnapshotQueryService {
	return &SubdomainSnapshotQueryService{store: store, scanLookup: scanLookup}
}

func (service *SubdomainSnapshotQueryService) ListByScan(ctx context.Context, scanID, page, pageSize int, filter string) ([]snapshotdomain.SubdomainSnapshot, int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrSnapshotScanNotFound
		}
		return nil, 0, err
	}
	return service.store.FindByScanID(scanID, page, pageSize, filter)
}

func (service *SubdomainSnapshotQueryService) StreamByScan(ctx context.Context, scanID int) (*sql.Rows, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}
	return service.store.StreamByScanID(scanID)
}

func (service *SubdomainSnapshotQueryService) CountByScan(ctx context.Context, scanID int) (int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrSnapshotScanNotFound
		}
		return 0, err
	}
	return service.store.CountByScanID(scanID)
}

func (service *SubdomainSnapshotQueryService) ScanRow(ctx context.Context, rows *sql.Rows) (*snapshotdomain.SubdomainSnapshot, error) {
	_ = ctx
	return service.store.ScanRow(rows)
}

type SubdomainSnapshotCommandService struct {
	store      SubdomainSnapshotCommandStore
	scanLookup SnapshotScanRefLookup
	assetSync  SubdomainAssetSync
}

func NewSubdomainSnapshotCommandService(store SubdomainSnapshotCommandStore, scanLookup SnapshotScanRefLookup, assetSync SubdomainAssetSync) *SubdomainSnapshotCommandService {
	return &SubdomainSnapshotCommandService{store: store, scanLookup: scanLookup, assetSync: assetSync}
}

func (service *SubdomainSnapshotCommandService) SaveAndSync(ctx context.Context, scanID int, targetID int, items []SubdomainSnapshotItem) (snapshotCount int64, assetCount int64, err error) {
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
	if !snapshotdomain.IsDomainTargetType(target.Type) {
		return 0, 0, ErrSubdomainSnapshotInvalidTargetType
	}

	snapshots := make([]snapshotdomain.SubdomainSnapshot, 0, len(items))
	validNames := make([]string, 0, len(items))
	for _, item := range items {
		if snapshotdomain.IsSubdomainMatchTarget(item.Name, *target) {
			snapshots = append(snapshots, snapshotdomain.SubdomainSnapshot{ScanID: scanID, Name: item.Name})
			validNames = append(validNames, item.Name)
		}
	}

	if len(snapshots) == 0 {
		return 0, 0, nil
	}
	snapshotCount, err = service.store.BulkCreate(snapshots)
	if err != nil {
		return 0, 0, err
	}
	assetCountInt, err := service.assetSync.BulkCreate(targetID, validNames)
	if err != nil {
		return snapshotCount, 0, nil
	}
	return snapshotCount, int64(assetCountInt), nil
}
