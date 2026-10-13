package assetwiring

import (
	"context"
	"time"

	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
)

type assetGlobalExportGrantStoreAdapter struct {
	store *assetrepo.GlobalAssetExportGrantStore
}

func newAssetGlobalExportGrantStoreAdapter(store *assetrepo.GlobalAssetExportGrantStore) *assetGlobalExportGrantStoreAdapter {
	return &assetGlobalExportGrantStoreAdapter{store: store}
}

func (adapter *assetGlobalExportGrantStoreAdapter) Issue(ctx context.Context, grant assetapp.GlobalAssetExportGrant, ttl time.Duration) (string, error) {
	return adapter.store.Issue(ctx, grant, ttl)
}

func (adapter *assetGlobalExportGrantStoreAdapter) Consume(ctx context.Context, token string) (*assetapp.GlobalAssetExportGrant, error) {
	return adapter.store.Consume(ctx, token)
}

type assetGlobalExportSnapshotStoreAdapter struct {
	store *assetrepo.GlobalAssetExportSnapshotStore
}

func newAssetGlobalExportSnapshotStoreAdapter(store *assetrepo.GlobalAssetExportSnapshotStore) *assetGlobalExportSnapshotStoreAdapter {
	return &assetGlobalExportSnapshotStoreAdapter{store: store}
}

func (adapter *assetGlobalExportSnapshotStoreAdapter) WithSnapshot(ctx context.Context, ast assetapp.GlobalAssetSearchAST, assetType assetapp.GlobalAssetSearchAssetType, visit func(assetapp.GlobalAssetExportSnapshot) error) error {
	return adapter.store.WithSnapshot(ctx, ast, assetType, visit)
}
