package application

import (
	"context"
	"database/sql"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type WebsiteSnapshotQueryService struct {
	store      WebsiteSnapshotQueryStore
	scanLookup SnapshotScanRefLookup
}

func NewWebsiteSnapshotQueryService(store WebsiteSnapshotQueryStore, scanLookup SnapshotScanRefLookup) *WebsiteSnapshotQueryService {
	return &WebsiteSnapshotQueryService{store: store, scanLookup: scanLookup}
}

func (service *WebsiteSnapshotQueryService) ListByScan(ctx context.Context, scanID, page, pageSize int, filter string) ([]snapshotdomain.WebsiteSnapshot, int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrSnapshotScanNotFound
		}
		return nil, 0, err
	}
	return service.store.FindByScanID(scanID, page, pageSize, filter)
}

func (service *WebsiteSnapshotQueryService) StreamByScan(ctx context.Context, scanID int) (*sql.Rows, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}
	return service.store.StreamByScanID(scanID)
}

func (service *WebsiteSnapshotQueryService) CountByScan(ctx context.Context, scanID int) (int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrSnapshotScanNotFound
		}
		return 0, err
	}
	return service.store.CountByScanID(scanID)
}

func (service *WebsiteSnapshotQueryService) ScanRow(ctx context.Context, rows *sql.Rows) (*snapshotdomain.WebsiteSnapshot, error) {
	_ = ctx
	return service.store.ScanRow(rows)
}

type WebsiteSnapshotCommandService struct {
	store      WebsiteSnapshotCommandStore
	scanLookup SnapshotScanRefLookup
	assetSync  WebsiteAssetSync
}

func NewWebsiteSnapshotCommandService(store WebsiteSnapshotCommandStore, scanLookup SnapshotScanRefLookup, assetSync WebsiteAssetSync) *WebsiteSnapshotCommandService {
	return &WebsiteSnapshotCommandService{store: store, scanLookup: scanLookup, assetSync: assetSync}
}

func (service *WebsiteSnapshotCommandService) SaveAndSync(ctx context.Context, scanID int, targetID int, items []WebsiteSnapshotItem) (snapshotCount int64, assetCount int64, err error) {
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

	snapshots := make([]snapshotdomain.WebsiteSnapshot, 0, len(items))
	validItems := make([]WebsiteAssetUpsertItem, 0, len(items))
	for _, item := range items {
		if !snapshotdomain.IsURLMatchTarget(item.URL, *target) {
			continue
		}
		host := item.Host
		if host == "" {
			host = snapshotdomain.ExtractHostFromURL(item.URL)
		}
		snapshots = append(snapshots, snapshotdomain.WebsiteSnapshot{ScanID: scanID, URL: item.URL, Host: host, Title: item.Title, StatusCode: item.StatusCode, ContentLength: item.ContentLength, Location: item.Location, Webserver: item.Webserver, ContentType: item.ContentType, Tech: item.Tech, ResponseBody: item.ResponseBody, Vhost: item.Vhost, ResponseHeaders: item.ResponseHeaders})
		validItems = append(validItems, WebsiteAssetUpsertItem{URL: item.URL, Host: item.Host, Title: item.Title, StatusCode: item.StatusCode, ContentLength: item.ContentLength, Location: item.Location, Webserver: item.Webserver, ContentType: item.ContentType, Tech: item.Tech, ResponseBody: item.ResponseBody, Vhost: item.Vhost, ResponseHeaders: item.ResponseHeaders})
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
