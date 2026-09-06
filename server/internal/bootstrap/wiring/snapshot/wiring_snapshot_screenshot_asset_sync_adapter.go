package snapshotwiring

import (
	"context"

	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

type snapshotScreenshotAssetSyncAdapter struct {
	service *assetapp.ScreenshotFacade
}

func newSnapshotScreenshotAssetSyncAdapter(service *assetapp.ScreenshotFacade) *snapshotScreenshotAssetSyncAdapter {
	return &snapshotScreenshotAssetSyncAdapter{service: service}
}

func (adapter *snapshotScreenshotAssetSyncAdapter) BatchUpsert(targetID int, req *snapshotapp.ScreenshotAssetUpsertRequest) (int64, error) {
	return adapter.service.BatchUpsert(targetID, snapshotScreenshotAssetRequestToApplication(req))
}

func (adapter *snapshotScreenshotAssetSyncAdapter) BatchUpsertContext(ctx context.Context, targetID int, req *snapshotapp.ScreenshotAssetUpsertRequest) (int64, error) {
	return adapter.service.BatchUpsertContext(ctx, targetID, snapshotScreenshotAssetRequestToApplication(req))
}
