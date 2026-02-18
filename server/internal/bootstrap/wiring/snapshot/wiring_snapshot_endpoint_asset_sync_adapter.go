package snapshotwiring

import (
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

type snapshotEndpointAssetSyncAdapter struct {
	service *assetapp.EndpointFacade
}

func newSnapshotEndpointAssetSyncAdapter(service *assetapp.EndpointFacade) *snapshotEndpointAssetSyncAdapter {
	return &snapshotEndpointAssetSyncAdapter{service: service}
}

func (adapter *snapshotEndpointAssetSyncAdapter) BulkUpsert(targetID int, items []snapshotapp.EndpointAssetUpsertItem) (int64, error) {
	return adapter.service.BulkUpsert(targetID, snapshotEndpointAssetUpsertItemsToApplication(items))
}
