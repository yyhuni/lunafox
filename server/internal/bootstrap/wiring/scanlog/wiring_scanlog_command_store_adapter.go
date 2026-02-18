package scanlogwiring

import (
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type scanLogCommandStoreAdapter struct {
	repo *scanrepo.ScanLogRepository
}

func newScanLogCommandStoreAdapter(repo *scanrepo.ScanLogRepository) *scanLogCommandStoreAdapter {
	return &scanLogCommandStoreAdapter{repo: repo}
}

func (adapter *scanLogCommandStoreAdapter) BulkCreate(logs []scanapp.ScanLogEntry) error {
	return adapter.repo.BulkCreate(logs)
}
