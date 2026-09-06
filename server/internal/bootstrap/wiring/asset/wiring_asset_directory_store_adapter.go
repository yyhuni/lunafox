package assetwiring

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
)

type assetDirectoryStoreAdapter struct {
	repo *assetrepo.DirectoryRepository
}

func newAssetDirectoryStoreAdapter(repo *assetrepo.DirectoryRepository) *assetDirectoryStoreAdapter {
	return &assetDirectoryStoreAdapter{repo: repo}
}

func (adapter *assetDirectoryStoreAdapter) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Directory, int64, error) {
	return adapter.repo.ListByTargetID(targetID, page, pageSize, filter, orderBy)
}

func (adapter *assetDirectoryStoreAdapter) ListByTargetIDContext(ctx context.Context, targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Directory, int64, error) {
	return adapter.repo.ListByTargetIDContext(ctx, targetID, page, pageSize, filter, orderBy)
}

func (adapter *assetDirectoryStoreAdapter) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	return adapter.repo.ListFilterOptionsByTargetID(targetID, field)
}

func (adapter *assetDirectoryStoreAdapter) BatchCreate(items []assetdomain.Directory) (int, error) {
	return adapter.repo.BatchCreate(items)
}

func (adapter *assetDirectoryStoreAdapter) BatchDelete(ids []int) (int64, error) {
	return adapter.repo.BatchDelete(ids)
}

func (adapter *assetDirectoryStoreAdapter) BatchUpsert(items []assetdomain.Directory) (int64, error) {
	return adapter.repo.BatchUpsert(items)
}

func (adapter *assetDirectoryStoreAdapter) BatchUpsertContext(ctx context.Context, items []assetdomain.Directory) (int64, error) {
	return adapter.repo.BatchUpsertContext(ctx, items)
}

func (adapter *assetDirectoryStoreAdapter) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Directory) error) error {
	return adapter.repo.ForEachByTargetID(ctx, targetID, visit)
}

func (adapter *assetDirectoryStoreAdapter) CountByTargetID(targetID int) (int64, error) {
	return adapter.repo.CountByTargetID(targetID)
}
