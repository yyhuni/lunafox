package assetwiring

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
)

type assetSubdomainStoreAdapter struct {
	repo *assetrepo.SubdomainRepository
}

func newAssetSubdomainStoreAdapter(repo *assetrepo.SubdomainRepository) *assetSubdomainStoreAdapter {
	return &assetSubdomainStoreAdapter{repo: repo}
}

func (adapter *assetSubdomainStoreAdapter) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Subdomain, int64, error) {
	return adapter.repo.ListByTargetID(targetID, page, pageSize, filter, orderBy)
}

func (adapter *assetSubdomainStoreAdapter) ListByTargetIDContext(ctx context.Context, targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Subdomain, int64, error) {
	return adapter.repo.ListByTargetIDContext(ctx, targetID, page, pageSize, filter, orderBy)
}

func (adapter *assetSubdomainStoreAdapter) BatchCreate(items []assetdomain.Subdomain) (int, error) {
	return adapter.repo.BatchCreate(items)
}

func (adapter *assetSubdomainStoreAdapter) BatchCreateContext(ctx context.Context, items []assetdomain.Subdomain) (int, error) {
	return adapter.repo.BatchCreateContext(ctx, items)
}

func (adapter *assetSubdomainStoreAdapter) BatchDelete(ids []int) (int64, error) {
	return adapter.repo.BatchDelete(ids)
}

func (adapter *assetSubdomainStoreAdapter) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Subdomain) error) error {
	return adapter.repo.ForEachByTargetID(ctx, targetID, visit)
}

func (adapter *assetSubdomainStoreAdapter) CountByTargetID(targetID int) (int64, error) {
	return adapter.repo.CountByTargetID(targetID)
}
