package scanwiring

import (
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type scanQueryStoreAdapter struct {
	repo *scanrepo.ScanRepository
}

func newScanQueryStoreAdapter(repo *scanrepo.ScanRepository) *scanQueryStoreAdapter {
	return &scanQueryStoreAdapter{repo: repo}
}

func (adapter *scanQueryStoreAdapter) FindAll(page, pageSize int, targetID int, status, search string) ([]scanapp.QueryScan, int64, error) {
	return adapter.repo.FindAll(page, pageSize, targetID, status, search)
}

func (adapter *scanQueryStoreAdapter) GetQueryByID(id int) (*scanapp.QueryScan, error) {
	return adapter.repo.GetQueryByID(id)
}

func (adapter *scanQueryStoreAdapter) GetStatistics() (*scanapp.QueryStatistics, error) {
	return adapter.repo.GetStatistics()
}
