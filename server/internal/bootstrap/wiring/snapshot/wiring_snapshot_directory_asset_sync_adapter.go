package snapshotwiring

import (
	"context"

	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

type snapshotDirectoryAssetSyncAdapter struct {
	service *assetapp.DirectoryFacade
}

func newSnapshotDirectoryAssetSyncAdapter(service *assetapp.DirectoryFacade) *snapshotDirectoryAssetSyncAdapter {
	return &snapshotDirectoryAssetSyncAdapter{service: service}
}

func (adapter *snapshotDirectoryAssetSyncAdapter) BatchUpsert(targetID int, items []snapshotapp.DirectoryAssetUpsertItem) (int64, error) {
	return adapter.service.BatchUpsert(targetID, snapshotDirectoryAssetUpsertItemsToApplication(items))
}

func (adapter *snapshotDirectoryAssetSyncAdapter) BatchUpsertContext(ctx context.Context, targetID int, items []snapshotapp.DirectoryAssetUpsertItem) (int64, error) {
	return adapter.service.BatchUpsertContext(ctx, targetID, snapshotDirectoryAssetUpsertItemsToApplication(items))
}
