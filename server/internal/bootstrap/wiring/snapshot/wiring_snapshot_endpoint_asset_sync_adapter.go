package snapshotwiring

import (
	"context"

	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

type snapshotEndpointAssetSyncAdapter struct {
	service *assetapp.EndpointFacade
}

func newSnapshotEndpointAssetSyncAdapter(service *assetapp.EndpointFacade) *snapshotEndpointAssetSyncAdapter {
	return &snapshotEndpointAssetSyncAdapter{service: service}
}

func (adapter *snapshotEndpointAssetSyncAdapter) BatchUpsert(targetID int, items []snapshotapp.EndpointAssetUpsertItem) (int64, error) {
	return adapter.service.BatchUpsert(targetID, snapshotEndpointAssetUpsertItemsToApplication(items))
}

func (adapter *snapshotEndpointAssetSyncAdapter) BatchUpsertContext(ctx context.Context, targetID int, items []snapshotapp.EndpointAssetUpsertItem) (int64, error) {
	return adapter.service.BatchUpsertContext(ctx, targetID, snapshotEndpointAssetUpsertItemsToApplication(items))
}
