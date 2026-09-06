package assetwiring

import (
	"context"

	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
)

type assetEndpointStoreAdapter struct {
	repo *assetrepo.EndpointRepository
}

func newAssetEndpointStoreAdapter(repo *assetrepo.EndpointRepository) *assetEndpointStoreAdapter {
	return &assetEndpointStoreAdapter{repo: repo}
}

func (adapter *assetEndpointStoreAdapter) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Endpoint, int64, error) {
	return adapter.repo.ListByTargetID(targetID, page, pageSize, filter, orderBy)
}

func (adapter *assetEndpointStoreAdapter) ListByTargetIDContext(ctx context.Context, targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Endpoint, int64, error) {
	return adapter.repo.ListByTargetIDContext(ctx, targetID, page, pageSize, filter, orderBy)
}

func (adapter *assetEndpointStoreAdapter) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	return adapter.repo.ListFilterOptionsByTargetID(targetID, field)
}

func (adapter *assetEndpointStoreAdapter) GetByID(id int) (*assetdomain.Endpoint, error) {
	return adapter.repo.GetByID(id)
}

func (adapter *assetEndpointStoreAdapter) BatchCreate(items []assetdomain.Endpoint) (int, error) {
	return adapter.repo.BatchCreate(items)
}

func (adapter *assetEndpointStoreAdapter) Delete(id int) error {
	return adapter.repo.Delete(id)
}

func (adapter *assetEndpointStoreAdapter) BatchDelete(ids []int) (int64, error) {
	return adapter.repo.BatchDelete(ids)
}

func (adapter *assetEndpointStoreAdapter) BatchUpsert(items []assetdomain.Endpoint) (int64, error) {
	return adapter.repo.BatchUpsert(items)
}

func (adapter *assetEndpointStoreAdapter) BatchUpsertContext(ctx context.Context, items []assetdomain.Endpoint) (int64, error) {
	return adapter.repo.BatchUpsertContext(ctx, items)
}

func (adapter *assetEndpointStoreAdapter) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Endpoint) error) error {
	return adapter.repo.ForEachByTargetID(ctx, targetID, visit)
}

func (adapter *assetEndpointStoreAdapter) CountByTargetID(targetID int) (int64, error) {
	return adapter.repo.CountByTargetID(targetID)
}

func (adapter *assetEndpointStoreAdapter) SearchGlobalEndpoints(ctx context.Context, query assetapp.GlobalAssetSearchStoreQuery) ([]assetdomain.Endpoint, error) {
	return adapter.repo.SearchGlobalEndpoints(ctx, query)
}
