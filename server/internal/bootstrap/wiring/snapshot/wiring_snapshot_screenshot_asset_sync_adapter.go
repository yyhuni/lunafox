package snapshotwiring

import (
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

type snapshotScreenshotAssetSyncAdapter struct {
	service *assetapp.ScreenshotFacade
}

func newSnapshotScreenshotAssetSyncAdapter(service *assetapp.ScreenshotFacade) *snapshotScreenshotAssetSyncAdapter {
	return &snapshotScreenshotAssetSyncAdapter{service: service}
}

func (adapter *snapshotScreenshotAssetSyncAdapter) BulkUpsert(targetID int, req *snapshotapp.ScreenshotAssetUpsertRequest) (int64, error) {
	return adapter.service.BulkUpsert(targetID, snapshotScreenshotAssetRequestToApplication(req))
}
