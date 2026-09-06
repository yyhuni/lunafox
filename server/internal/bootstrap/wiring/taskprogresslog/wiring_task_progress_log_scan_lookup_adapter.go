package taskprogresslogwiring

import (
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type taskProgressLogScanLookupAdapter struct {
	repo *scanrepo.ScanRepository
}

func newTaskProgressLogScanLookupAdapter(repo *scanrepo.ScanRepository) *taskProgressLogScanLookupAdapter {
	return &taskProgressLogScanLookupAdapter{repo: repo}
}

func (adapter *taskProgressLogScanLookupAdapter) GetTaskProgressLogRefByID(id int) (*scanapp.TaskProgressLogScanRef, error) {
	return adapter.repo.FindTaskProgressLogScanRefByID(id)
}
