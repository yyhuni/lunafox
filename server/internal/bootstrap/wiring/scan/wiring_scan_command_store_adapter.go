package scanwiring

import (
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type scanCommandStoreAdapter struct {
	repo *scanrepo.ScanRepository
}

func newScanCommandStoreAdapter(repo *scanrepo.ScanRepository) *scanCommandStoreAdapter {
	return &scanCommandStoreAdapter{repo: repo}
}

func (adapter *scanCommandStoreAdapter) GetByIDNotDeleted(id int) (*scanapp.QueryScan, error) {
	return adapter.repo.GetByIDNotDeleted(id)
}

func (adapter *scanCommandStoreAdapter) FindByIDs(ids []int) ([]scanapp.QueryScan, error) {
	return adapter.repo.FindByIDs(ids)
}

func (adapter *scanCommandStoreAdapter) CreateWithInputTargetsAndTasks(scan *scanapp.CreateScan, inputs []scanapp.CreateScanInputTarget, tasks []scanapp.CreateScanTask) error {
	return adapter.repo.CreateWithInputTargetsAndTasks(scan, inputs, tasks)
}

func (adapter *scanCommandStoreAdapter) BulkSoftDelete(ids []int) (int64, []string, error) {
	return adapter.repo.BulkSoftDelete(ids)
}

func (adapter *scanCommandStoreAdapter) UpdateStatus(id int, status string, errorMessage ...string) error {
	return adapter.repo.UpdateStatus(id, status, errorMessage...)
}
