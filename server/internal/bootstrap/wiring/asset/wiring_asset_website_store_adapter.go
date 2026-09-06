package assetwiring

import (
	"context"

	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
)

type assetWebsiteStoreAdapter struct {
	repo *assetrepo.WebsiteRepository
}

func newAssetWebsiteStoreAdapter(repo *assetrepo.WebsiteRepository) *assetWebsiteStoreAdapter {
	return &assetWebsiteStoreAdapter{repo: repo}
}

func (adapter *assetWebsiteStoreAdapter) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Website, int64, error) {
	return adapter.repo.ListByTargetID(targetID, page, pageSize, filter, orderBy)
}

func (adapter *assetWebsiteStoreAdapter) ListByTargetIDContext(ctx context.Context, targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Website, int64, error) {
	return adapter.repo.ListByTargetIDContext(ctx, targetID, page, pageSize, filter, orderBy)
}

func (adapter *assetWebsiteStoreAdapter) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	return adapter.repo.ListFilterOptionsByTargetID(targetID, field)
}

func (adapter *assetWebsiteStoreAdapter) GetByID(id int) (*assetdomain.Website, error) {
	return adapter.repo.GetByID(id)
}

func (adapter *assetWebsiteStoreAdapter) BatchCreate(items []assetdomain.Website) (int, error) {
	return adapter.repo.BatchCreate(items)
}

func (adapter *assetWebsiteStoreAdapter) BatchCreateContext(ctx context.Context, items []assetdomain.Website) (int, error) {
	return adapter.repo.BatchCreateContext(ctx, items)
}

func (adapter *assetWebsiteStoreAdapter) Delete(id int) error {
	return adapter.repo.Delete(id)
}

func (adapter *assetWebsiteStoreAdapter) BatchDelete(ids []int) (int64, error) {
	return adapter.repo.BatchDelete(ids)
}

func (adapter *assetWebsiteStoreAdapter) BatchUpsert(items []assetdomain.Website) (int64, error) {
	return adapter.repo.BatchUpsert(items)
}

func (adapter *assetWebsiteStoreAdapter) BatchUpsertContext(ctx context.Context, items []assetdomain.Website) (int64, error) {
	return adapter.repo.BatchUpsertContext(ctx, items)
}

func (adapter *assetWebsiteStoreAdapter) BatchUpsertTechnologyContext(ctx context.Context, targetID int, items []assetdomain.WebsiteTechnology) (int64, error) {
	return adapter.repo.BatchUpsertTechnologyContext(ctx, targetID, items)
}

func (adapter *assetWebsiteStoreAdapter) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Website) error) error {
	return adapter.repo.ForEachByTargetID(ctx, targetID, visit)
}

func (adapter *assetWebsiteStoreAdapter) CountByTargetID(targetID int) (int64, error) {
	return adapter.repo.CountByTargetID(targetID)
}

func (adapter *assetWebsiteStoreAdapter) SearchGlobalWebsites(ctx context.Context, query assetapp.GlobalAssetSearchStoreQuery) ([]assetdomain.Website, error) {
	return adapter.repo.SearchGlobalWebsites(ctx, query)
}
