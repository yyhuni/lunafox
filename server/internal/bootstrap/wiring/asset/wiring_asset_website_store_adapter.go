package assetwiring

import (
	"database/sql"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
)

type assetWebsiteStoreAdapter struct {
	repo *assetrepo.WebsiteRepository
}

func newAssetWebsiteStoreAdapter(repo *assetrepo.WebsiteRepository) *assetWebsiteStoreAdapter {
	return &assetWebsiteStoreAdapter{repo: repo}
}

func (adapter *assetWebsiteStoreAdapter) FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Website, int64, error) {
	return adapter.repo.FindByTargetID(targetID, page, pageSize, filter)
}

func (adapter *assetWebsiteStoreAdapter) GetByID(id int) (*assetdomain.Website, error) {
	return adapter.repo.GetByID(id)
}

func (adapter *assetWebsiteStoreAdapter) BulkCreate(items []assetdomain.Website) (int, error) {
	return adapter.repo.BulkCreate(items)
}

func (adapter *assetWebsiteStoreAdapter) Delete(id int) error {
	return adapter.repo.Delete(id)
}

func (adapter *assetWebsiteStoreAdapter) BulkDelete(ids []int) (int64, error) {
	return adapter.repo.BulkDelete(ids)
}

func (adapter *assetWebsiteStoreAdapter) BulkUpsert(items []assetdomain.Website) (int64, error) {
	return adapter.repo.BulkUpsert(items)
}

func (adapter *assetWebsiteStoreAdapter) StreamByTargetID(targetID int) (*sql.Rows, error) {
	return adapter.repo.StreamByTargetID(targetID)
}

func (adapter *assetWebsiteStoreAdapter) CountByTargetID(targetID int) (int64, error) {
	return adapter.repo.CountByTargetID(targetID)
}

func (adapter *assetWebsiteStoreAdapter) ScanRow(rows *sql.Rows) (*assetdomain.Website, error) {
	return adapter.repo.ScanRow(rows)
}
