package scanlogwiring

import (
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type scanLogQueryStoreAdapter struct {
	repo *scanrepo.ScanLogRepository
}

func newScanLogQueryStoreAdapter(repo *scanrepo.ScanLogRepository) *scanLogQueryStoreAdapter {
	return &scanLogQueryStoreAdapter{repo: repo}
}

func (adapter *scanLogQueryStoreAdapter) FindByScanIDWithCursor(scanID int, afterID int64, limit int) ([]scanapp.ScanLogEntry, error) {
	return adapter.repo.FindByScanIDWithCursor(scanID, afterID, limit)
}
