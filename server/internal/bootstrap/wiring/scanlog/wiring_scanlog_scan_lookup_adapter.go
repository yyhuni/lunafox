package scanlogwiring

import (
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type scanLogScanLookupAdapter struct {
	repo *scanrepo.ScanRepository
}

func newScanLogScanLookupAdapter(repo *scanrepo.ScanRepository) *scanLogScanLookupAdapter {
	return &scanLogScanLookupAdapter{repo: repo}
}

func (adapter *scanLogScanLookupAdapter) GetScanLogRefByID(id int) (*scanapp.ScanLogScanRef, error) {
	return adapter.repo.FindScanLogScanRefByID(id)
}
