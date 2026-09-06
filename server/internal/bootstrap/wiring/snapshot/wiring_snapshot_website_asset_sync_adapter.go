package snapshotwiring

import (
	"context"

	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

type snapshotWebsiteAssetSyncAdapter struct {
	service *assetapp.WebsiteFacade
}

func newSnapshotWebsiteAssetSyncAdapter(service *assetapp.WebsiteFacade) *snapshotWebsiteAssetSyncAdapter {
	return &snapshotWebsiteAssetSyncAdapter{service: service}
}

func (adapter *snapshotWebsiteAssetSyncAdapter) BatchUpsertContext(ctx context.Context, targetID int, items []snapshotapp.WebsiteAssetUpsertItem) (int64, error) {
	return adapter.service.BatchUpsertContext(ctx, targetID, snapshotWebsiteAssetUpsertItemsToApplication(items))
}
