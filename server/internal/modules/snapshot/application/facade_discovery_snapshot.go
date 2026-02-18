package application

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

// DirectorySnapshotFacade handles directory snapshot business logic.
type DirectorySnapshotFacade struct {
	queryService *DirectorySnapshotQueryService
	cmdService   *DirectorySnapshotCommandService
}

// SubdomainSnapshotFacade handles subdomain snapshot business logic.
type SubdomainSnapshotFacade struct {
	queryService *SubdomainSnapshotQueryService
	cmdService   *SubdomainSnapshotCommandService
}

// NewDirectorySnapshotFacade creates a new directory snapshot service.
func NewDirectorySnapshotFacade(
	queryService *DirectorySnapshotQueryService,
	cmdService *DirectorySnapshotCommandService,
) *DirectorySnapshotFacade {
	return &DirectorySnapshotFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// NewSubdomainSnapshotFacade creates a new subdomain snapshot service.
func NewSubdomainSnapshotFacade(
	queryService *SubdomainSnapshotQueryService,
	cmdService *SubdomainSnapshotCommandService,
) *SubdomainSnapshotFacade {
	return &SubdomainSnapshotFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// SaveAndSync saves directory snapshots and syncs to asset table.
func (service *DirectorySnapshotFacade) SaveAndSync(scanID int, targetID int, items []DirectorySnapshotItem) (snapshotCount int64, assetCount int64, err error) {
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

// ListByScan returns paginated directory snapshots for a scan.
func (service *DirectorySnapshotFacade) ListByScan(scanID int, query *SnapshotListQuery) ([]DirectorySnapshot, int64, error) {
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
func (service *DirectorySnapshotFacade) StreamByScan(scanID int) (*sql.Rows, error) {
	rows, err := service.queryService.StreamByScan(context.Background(), scanID)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, ErrScanNotFoundForSnapshot
		}
		return nil, err
	}
	return rows, nil
}

// CountByScan returns the count of directory snapshots for a scan.
func (service *DirectorySnapshotFacade) CountByScan(scanID int) (int64, error) {
	count, err := service.queryService.CountByScan(context.Background(), scanID)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrScanNotFoundForSnapshot
		}
		return 0, err
	}
	return count, nil
}

// ScanRow scans a row into DirectorySnapshot model.
func (service *DirectorySnapshotFacade) ScanRow(rows *sql.Rows) (*DirectorySnapshot, error) {
	item, err := service.queryService.ScanRow(context.Background(), rows)
	if err != nil {
		return nil, err
	}
	return item, nil
}

// SaveAndSync saves subdomain snapshots and syncs to asset table.
func (service *SubdomainSnapshotFacade) SaveAndSync(scanID int, targetID int, items []SubdomainSnapshotItem) (snapshotCount int64, assetCount int64, err error) {
	snapshotCount, assetCount, err = service.cmdService.SaveAndSync(context.Background(), scanID, targetID, items)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, 0, ErrScanNotFoundForSnapshot
		}
		if errors.Is(err, ErrSnapshotTargetMismatch) {
			return 0, 0, ErrTargetMismatch
		}
		if errors.Is(err, ErrSubdomainSnapshotInvalidTargetType) {
			return 0, 0, ErrInvalidTargetType
		}
		return 0, 0, err
	}
	return snapshotCount, assetCount, nil
}

// ListByScan returns paginated subdomain snapshots for a scan.
func (service *SubdomainSnapshotFacade) ListByScan(scanID int, query *SnapshotListQuery) ([]SubdomainSnapshot, int64, error) {
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
func (service *SubdomainSnapshotFacade) StreamByScan(scanID int) (*sql.Rows, error) {
	rows, err := service.queryService.StreamByScan(context.Background(), scanID)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, ErrScanNotFoundForSnapshot
		}
		return nil, err
	}
	return rows, nil
}

// CountByScan returns the count of subdomain snapshots for a scan.
func (service *SubdomainSnapshotFacade) CountByScan(scanID int) (int64, error) {
	count, err := service.queryService.CountByScan(context.Background(), scanID)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrScanNotFoundForSnapshot
		}
		return 0, err
	}
	return count, nil
}

// ScanRow scans a row into SubdomainSnapshot model.
func (service *SubdomainSnapshotFacade) ScanRow(rows *sql.Rows) (*SubdomainSnapshot, error) {
	item, err := service.queryService.ScanRow(context.Background(), rows)
	if err != nil {
		return nil, err
	}
	return item, nil
}
