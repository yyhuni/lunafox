package application

import (
	"context"
	"database/sql"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type EndpointSnapshotQueryService struct {
	store      EndpointSnapshotQueryStore
	scanLookup SnapshotScanRefLookup
}

func NewEndpointSnapshotQueryService(store EndpointSnapshotQueryStore, scanLookup SnapshotScanRefLookup) *EndpointSnapshotQueryService {
	return &EndpointSnapshotQueryService{store: store, scanLookup: scanLookup}
}

func (service *EndpointSnapshotQueryService) ListByScan(ctx context.Context, scanID, page, pageSize int, filter string) ([]snapshotdomain.EndpointSnapshot, int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrSnapshotScanNotFound
		}
		return nil, 0, err
	}
	return service.store.FindByScanID(scanID, page, pageSize, filter)
}

func (service *EndpointSnapshotQueryService) StreamByScan(ctx context.Context, scanID int) (*sql.Rows, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}
	return service.store.StreamByScanID(scanID)
}

func (service *EndpointSnapshotQueryService) CountByScan(ctx context.Context, scanID int) (int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrSnapshotScanNotFound
		}
		return 0, err
	}
	return service.store.CountByScanID(scanID)
}

func (service *EndpointSnapshotQueryService) ScanRow(ctx context.Context, rows *sql.Rows) (*snapshotdomain.EndpointSnapshot, error) {
	_ = ctx
	return service.store.ScanRow(rows)
}

type EndpointSnapshotCommandService struct {
	store      EndpointSnapshotCommandStore
	scanLookup SnapshotScanRefLookup
	assetSync  EndpointAssetSync
}

func NewEndpointSnapshotCommandService(store EndpointSnapshotCommandStore, scanLookup SnapshotScanRefLookup, assetSync EndpointAssetSync) *EndpointSnapshotCommandService {
	return &EndpointSnapshotCommandService{store: store, scanLookup: scanLookup, assetSync: assetSync}
}

func (service *EndpointSnapshotCommandService) SaveAndSync(ctx context.Context, scanID int, targetID int, items []EndpointSnapshotItem) (snapshotCount int64, assetCount int64, err error) {
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

	snapshots := make([]snapshotdomain.EndpointSnapshot, 0, len(items))
	validItems := make([]EndpointAssetUpsertItem, 0, len(items))
	for _, item := range items {
		if !snapshotdomain.IsURLMatchTarget(item.URL, *target) {
			continue
		}
		host := item.Host
		if host == "" {
			host = snapshotdomain.ExtractHostFromURL(item.URL)
		}
		snapshots = append(snapshots, snapshotdomain.EndpointSnapshot{ScanID: scanID, URL: item.URL, Host: host, Title: item.Title, StatusCode: item.StatusCode, ContentLength: item.ContentLength, Location: item.Location, Webserver: item.Webserver, ContentType: item.ContentType, Tech: item.Tech, ResponseBody: item.ResponseBody, Vhost: item.Vhost, ResponseHeaders: item.ResponseHeaders})
		validItems = append(validItems, EndpointAssetUpsertItem{URL: item.URL, Host: item.Host, Title: item.Title, StatusCode: item.StatusCode, ContentLength: item.ContentLength, Location: item.Location, Webserver: item.Webserver, ContentType: item.ContentType, Tech: item.Tech, ResponseBody: item.ResponseBody, Vhost: item.Vhost, ResponseHeaders: item.ResponseHeaders})
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
