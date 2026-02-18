package scanwiring

import (
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type scanTaskRuntimeStoreAdapter struct{ repo *scanrepo.ScanRepository }

func newScanTaskRuntimeStoreAdapter(repo *scanrepo.ScanRepository) *scanTaskRuntimeStoreAdapter {
	return &scanTaskRuntimeStoreAdapter{repo: repo}
}

func (adapter *scanTaskRuntimeStoreAdapter) GetTaskRuntimeByID(id int) (*scanapp.TaskScanRecord, error) {
	return adapter.repo.GetTaskRuntimeByID(id)
}

func (adapter *scanTaskRuntimeStoreAdapter) UpdateStatus(id int, status string, errorMessage string) error {
	if errorMessage == "" {
		return adapter.repo.UpdateStatus(id, status)
	}
	return adapter.repo.UpdateStatus(id, status, errorMessage)
}
