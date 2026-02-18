package application

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

// WebsiteSnapshotFacade handles website snapshot business logic.
type WebsiteSnapshotFacade struct {
	queryService *WebsiteSnapshotQueryService
	cmdService   *WebsiteSnapshotCommandService
}

// EndpointSnapshotFacade handles endpoint snapshot business logic.
type EndpointSnapshotFacade struct {
	queryService *EndpointSnapshotQueryService
	cmdService   *EndpointSnapshotCommandService
}

// NewWebsiteSnapshotFacade creates a new website snapshot service.
func NewWebsiteSnapshotFacade(
	queryService *WebsiteSnapshotQueryService,
	cmdService *WebsiteSnapshotCommandService,
) *WebsiteSnapshotFacade {
	return &WebsiteSnapshotFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// NewEndpointSnapshotFacade creates a new endpoint snapshot service.
func NewEndpointSnapshotFacade(
	queryService *EndpointSnapshotQueryService,
	cmdService *EndpointSnapshotCommandService,
) *EndpointSnapshotFacade {
	return &EndpointSnapshotFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// SaveAndSync saves website snapshots and syncs to asset table.
func (service *WebsiteSnapshotFacade) SaveAndSync(scanID int, targetID int, items []WebsiteSnapshotItem) (snapshotCount int64, assetCount int64, err error) {
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

// ListByScan returns paginated website snapshots for a scan.
func (service *WebsiteSnapshotFacade) ListByScan(scanID int, query *SnapshotListQuery) ([]WebsiteSnapshot, int64, error) {
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
func (service *WebsiteSnapshotFacade) StreamByScan(scanID int) (*sql.Rows, error) {
	rows, err := service.queryService.StreamByScan(context.Background(), scanID)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, ErrScanNotFoundForSnapshot
		}
		return nil, err
	}
	return rows, nil
}

// CountByScan returns the count of website snapshots for a scan.
func (service *WebsiteSnapshotFacade) CountByScan(scanID int) (int64, error) {
	count, err := service.queryService.CountByScan(context.Background(), scanID)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrScanNotFoundForSnapshot
		}
		return 0, err
	}
	return count, nil
}

// ScanRow scans a row into WebsiteSnapshot model.
func (service *WebsiteSnapshotFacade) ScanRow(rows *sql.Rows) (*WebsiteSnapshot, error) {
	item, err := service.queryService.ScanRow(context.Background(), rows)
	if err != nil {
		return nil, err
	}
	return item, nil
}

// SaveAndSync saves endpoint snapshots and syncs to asset table.
func (service *EndpointSnapshotFacade) SaveAndSync(scanID int, targetID int, items []EndpointSnapshotItem) (snapshotCount int64, assetCount int64, err error) {
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

// ListByScan returns paginated endpoint snapshots for a scan.
func (service *EndpointSnapshotFacade) ListByScan(scanID int, query *SnapshotListQuery) ([]EndpointSnapshot, int64, error) {
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
func (service *EndpointSnapshotFacade) StreamByScan(scanID int) (*sql.Rows, error) {
	rows, err := service.queryService.StreamByScan(context.Background(), scanID)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, ErrScanNotFoundForSnapshot
		}
		return nil, err
	}
	return rows, nil
}

// CountByScan returns the count of endpoint snapshots for a scan.
func (service *EndpointSnapshotFacade) CountByScan(scanID int) (int64, error) {
	count, err := service.queryService.CountByScan(context.Background(), scanID)
	if err != nil {
		if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrScanNotFoundForSnapshot
		}
		return 0, err
	}
	return count, nil
}

// ScanRow scans a row into EndpointSnapshot model.
func (service *EndpointSnapshotFacade) ScanRow(rows *sql.Rows) (*EndpointSnapshot, error) {
	item, err := service.queryService.ScanRow(context.Background(), rows)
	if err != nil {
		return nil, err
	}
	return item, nil
}
