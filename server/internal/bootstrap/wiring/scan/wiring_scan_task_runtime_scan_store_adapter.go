package scanwiring

import (
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type scanTaskRuntimeScanStoreAdapter struct{ repo *scanrepo.ScanRepository }

func newScanTaskRuntimeScanStoreAdapter(repo *scanrepo.ScanRepository) *scanTaskRuntimeScanStoreAdapter {
	return &scanTaskRuntimeScanStoreAdapter{repo: repo}
}

func (adapter *scanTaskRuntimeScanStoreAdapter) GetScanForScanTask(scanID int) (*scanapp.ScanTaskRuntimeScanRecord, error) {
	return adapter.repo.GetScanForScanTask(scanID)
}

func (adapter *scanTaskRuntimeScanStoreAdapter) UpdateScanStatus(id int, status string, failure *scanapp.FailureDetail) error {
	return adapter.repo.UpdateScanStatus(id, status, failure)
}
