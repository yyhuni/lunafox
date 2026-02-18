package snapshotwiring

import (
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

type snapshotWebsiteAssetSyncAdapter struct {
	service *assetapp.WebsiteFacade
}

func newSnapshotWebsiteAssetSyncAdapter(service *assetapp.WebsiteFacade) *snapshotWebsiteAssetSyncAdapter {
	return &snapshotWebsiteAssetSyncAdapter{service: service}
}

func (adapter *snapshotWebsiteAssetSyncAdapter) BulkUpsert(targetID int, items []snapshotapp.WebsiteAssetUpsertItem) (int64, error) {
	return adapter.service.BulkUpsert(targetID, snapshotWebsiteAssetUpsertItemsToApplication(items))
}
