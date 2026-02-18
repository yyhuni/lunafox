package assetwiring

import (
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
)

type assetScreenshotStoreAdapter struct {
	repo *assetrepo.ScreenshotRepository
}

func newAssetScreenshotStoreAdapter(repo *assetrepo.ScreenshotRepository) *assetScreenshotStoreAdapter {
	return &assetScreenshotStoreAdapter{repo: repo}
}

func (adapter *assetScreenshotStoreAdapter) FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Screenshot, int64, error) {
	return adapter.repo.FindByTargetID(targetID, page, pageSize, filter)
}

func (adapter *assetScreenshotStoreAdapter) GetByID(id int) (*assetdomain.Screenshot, error) {
	return adapter.repo.GetByID(id)
}

func (adapter *assetScreenshotStoreAdapter) BulkDelete(ids []int) (int64, error) {
	return adapter.repo.BulkDelete(ids)
}

func (adapter *assetScreenshotStoreAdapter) BulkUpsert(items []assetdomain.Screenshot) (int64, error) {
	return adapter.repo.BulkUpsert(items)
}
