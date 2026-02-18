package application

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

// HostPortSnapshotFacade handles host-port snapshot business logic.
type HostPortSnapshotFacade struct {
	queryService *HostPortSnapshotQueryService
	cmdService   *HostPortSnapshotCommandService
}

// ScreenshotSnapshotFacade handles screenshot snapshot business logic.
type ScreenshotSnapshotFacade struct {
	queryService *ScreenshotSnapshotQueryService
	cmdService   *ScreenshotSnapshotCommandService
}

// NewHostPortSnapshotFacade creates a new host-port snapshot service.
func NewHostPortSnapshotFacade(
	queryService *HostPortSnapshotQueryService,
	cmdService *HostPortSnapshotCommandService,
) *HostPortSnapshotFacade {
	return &HostPortSnapshotFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// NewScreenshotSnapshotFacade creates a new screenshot snapshot service.
func NewScreenshotSnapshotFacade(
	queryService *ScreenshotSnapshotQueryService,
	cmdService *ScreenshotSnapshotCommandService,
) *ScreenshotSnapshotFacade {
	return &ScreenshotSnapshotFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// SaveAndSync saves host-port snapshots and syncs to asset table.
func (service *HostPortSnapshotFacade) SaveAndSync(scanID int, targetID int, items []HostPortSnapshotItem) (snapshotCount int64, assetCount int64, err error) {
	snapshotCount, assetCount, err = service.cmdService.SaveAndSync(context.Background(), scanID, targetID, items)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, 0, ErrScanNotFoundForSnapshot
		}
		if errors.Is(err, ErrSnapshotTargetMismatch) {
			return 0, 0, ErrTargetMismatch
		}
		return 0, 0, err
	}
	return snapshotCount, assetCount, nil
}

// ListByScan returns paginated host-port snapshots for a scan.
func (service *HostPortSnapshotFacade) ListByScan(scanID int, query *SnapshotListQuery) ([]HostPortSnapshot, int64, error) {
	page, pageSize, filter := query.normalize()
	items, total, err := service.queryService.ListByScan(context.Background(), scanID, page, pageSize, filter)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrScanNotFoundForSnapshot
		}
		return nil, 0, err
	}
	return items, total, nil
}

// StreamByScan returns a cursor for streaming export.
func (service *HostPortSnapshotFacade) StreamByScan(scanID int) (*sql.Rows, error) {
	rows, err := service.queryService.StreamByScan(context.Background(), scanID)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, ErrScanNotFoundForSnapshot
		}
		return nil, err
	}
	return rows, nil
}

// CountByScan returns the count of host-port snapshots for a scan.
func (service *HostPortSnapshotFacade) CountByScan(scanID int) (int64, error) {
	count, err := service.queryService.CountByScan(context.Background(), scanID)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrScanNotFoundForSnapshot
		}
		return 0, err
	}
	return count, nil
}

// ScanRow scans a row into HostPortSnapshot model.
func (service *HostPortSnapshotFacade) ScanRow(rows *sql.Rows) (*HostPortSnapshot, error) {
	item, err := service.queryService.ScanRow(context.Background(), rows)
	if err != nil {
		return nil, err
	}
	return item, nil
}

// SaveAndSync saves screenshot snapshots and syncs to asset table.
func (service *ScreenshotSnapshotFacade) SaveAndSync(scanID int, targetID int, items []ScreenshotSnapshotItem) (snapshotCount int64, assetCount int64, err error) {
	snapshotCount, assetCount, err = service.cmdService.SaveAndSync(context.Background(), scanID, targetID, items)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, 0, ErrScanNotFoundForSnapshot
		}
		if errors.Is(err, ErrSnapshotTargetMismatch) {
			return 0, 0, ErrTargetMismatch
		}
		return 0, 0, err
	}
	return snapshotCount, assetCount, nil
}

// ListByScan returns paginated screenshot snapshots for a scan.
func (service *ScreenshotSnapshotFacade) ListByScan(scanID int, query *SnapshotListQuery) ([]ScreenshotSnapshot, int64, error) {
	page, pageSize, filter := query.normalize()
	items, total, err := service.queryService.ListByScan(context.Background(), scanID, page, pageSize, filter)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrScanNotFoundForSnapshot
		}
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID returns a screenshot snapshot by ID under a scan (including image data).
func (service *ScreenshotSnapshotFacade) GetByID(scanID int, id int) (*ScreenshotSnapshot, error) {
	item, err := service.queryService.GetByID(context.Background(), scanID, id)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, ErrScanNotFoundForSnapshot
		}
		if errors.Is(err, ErrScreenshotSnapshotNotFound) {
			return nil, ErrScreenshotSnapshotNotFound
		}
		return nil, err
	}
	return item, nil
}
