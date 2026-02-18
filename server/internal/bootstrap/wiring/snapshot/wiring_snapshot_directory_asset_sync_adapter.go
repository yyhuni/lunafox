package snapshotwiring

import (
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

type snapshotDirectoryAssetSyncAdapter struct {
	service *assetapp.DirectoryFacade
}

func newSnapshotDirectoryAssetSyncAdapter(service *assetapp.DirectoryFacade) *snapshotDirectoryAssetSyncAdapter {
	return &snapshotDirectoryAssetSyncAdapter{service: service}
}

func (adapter *snapshotDirectoryAssetSyncAdapter) BulkUpsert(targetID int, items []snapshotapp.DirectoryAssetUpsertItem) (int64, error) {
	return adapter.service.BulkUpsert(targetID, snapshotDirectoryAssetUpsertItemsToApplication(items))
}
