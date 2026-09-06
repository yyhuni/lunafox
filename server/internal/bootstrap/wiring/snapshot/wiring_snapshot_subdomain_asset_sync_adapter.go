package snapshotwiring

import (
	"context"

	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

type snapshotSubdomainAssetSyncAdapter struct {
	service *assetapp.SubdomainFacade
}

func newSnapshotSubdomainAssetSyncAdapter(service *assetapp.SubdomainFacade) *snapshotSubdomainAssetSyncAdapter {
	return &snapshotSubdomainAssetSyncAdapter{service: service}
}

func (adapter *snapshotSubdomainAssetSyncAdapter) BatchCreateContext(ctx context.Context, targetID int, dnsNames []string) (int, error) {
	return adapter.service.BatchCreateContext(ctx, targetID, dnsNames)
}
