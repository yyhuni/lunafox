package snapshotwiring

import (
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

type snapshotHostPortAssetSyncAdapter struct {
	service *assetapp.HostPortFacade
}

func newSnapshotHostPortAssetSyncAdapter(service *assetapp.HostPortFacade) *snapshotHostPortAssetSyncAdapter {
	return &snapshotHostPortAssetSyncAdapter{service: service}
}

func (adapter *snapshotHostPortAssetSyncAdapter) BulkUpsert(targetID int, items []snapshotapp.HostPortAssetItem) (int64, error) {
	return adapter.service.BulkUpsert(targetID, snapshotHostPortAssetItemsToApplication(items))
}
