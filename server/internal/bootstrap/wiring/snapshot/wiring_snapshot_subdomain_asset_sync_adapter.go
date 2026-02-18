package snapshotwiring

import assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"

type snapshotSubdomainAssetSyncAdapter struct {
	service *assetapp.SubdomainFacade
}

func newSnapshotSubdomainAssetSyncAdapter(service *assetapp.SubdomainFacade) *snapshotSubdomainAssetSyncAdapter {
	return &snapshotSubdomainAssetSyncAdapter{service: service}
}

func (adapter *snapshotSubdomainAssetSyncAdapter) BulkCreate(targetID int, names []string) (int, error) {
	return adapter.service.BulkCreate(targetID, names)
}
