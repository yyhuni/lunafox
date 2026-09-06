package assetwiring

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
)

type assetScreenshotStoreAdapter struct {
	repo *assetrepo.ScreenshotRepository
}

func newAssetScreenshotStoreAdapter(repo *assetrepo.ScreenshotRepository) *assetScreenshotStoreAdapter {
	return &assetScreenshotStoreAdapter{repo: repo}
}

func (adapter *assetScreenshotStoreAdapter) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Screenshot, int64, error) {
	return adapter.repo.ListByTargetID(targetID, page, pageSize, filter, orderBy)
}

func (adapter *assetScreenshotStoreAdapter) ListByTargetIDContext(ctx context.Context, targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Screenshot, int64, error) {
	return adapter.repo.ListByTargetIDContext(ctx, targetID, page, pageSize, filter, orderBy)
}

func (adapter *assetScreenshotStoreAdapter) ListSummariesByTargetAndURLs(targetID int, urls []string) ([]assetdomain.Screenshot, error) {
	return adapter.repo.ListSummariesByTargetAndURLs(targetID, urls)
}

func (adapter *assetScreenshotStoreAdapter) ListSummariesByTargetAndURLsContext(ctx context.Context, targetID int, urls []string) ([]assetdomain.Screenshot, error) {
	return adapter.repo.ListSummariesByTargetAndURLsContext(ctx, targetID, urls)
}

func (adapter *assetScreenshotStoreAdapter) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	return adapter.repo.ListFilterOptionsByTargetID(targetID, field)
}

func (adapter *assetScreenshotStoreAdapter) GetByID(id int) (*assetdomain.Screenshot, error) {
	return adapter.repo.GetByID(id)
}

func (adapter *assetScreenshotStoreAdapter) GetByIDContext(ctx context.Context, id int) (*assetdomain.Screenshot, error) {
	return adapter.repo.GetByIDContext(ctx, id)
}

func (adapter *assetScreenshotStoreAdapter) BatchDelete(ids []int) (int64, error) {
	return adapter.repo.BatchDelete(ids)
}

func (adapter *assetScreenshotStoreAdapter) BatchUpsert(items []assetdomain.Screenshot) (int64, error) {
	return adapter.repo.BatchUpsert(items)
}

func (adapter *assetScreenshotStoreAdapter) BatchUpsertContext(ctx context.Context, items []assetdomain.Screenshot) (int64, error) {
	return adapter.repo.BatchUpsertContext(ctx, items)
}
