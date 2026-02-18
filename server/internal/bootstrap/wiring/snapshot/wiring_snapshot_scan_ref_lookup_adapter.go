package snapshotwiring

import (
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type snapshotScanRefLookupAdapter struct {
	repo *scanrepo.ScanRepository
}

func newSnapshotScanRefLookupAdapter(repo *scanrepo.ScanRepository) *snapshotScanRefLookupAdapter {
	return &snapshotScanRefLookupAdapter{repo: repo}
}

func (adapter *snapshotScanRefLookupAdapter) GetScanRefByID(id int) (*snapshotdomain.ScanRef, error) {
	item, err := adapter.repo.GetByIDNotDeleted(id)
	if err != nil {
		return nil, err
	}
	return snapshotScanModelToDomain(item), nil
}

func (adapter *snapshotScanRefLookupAdapter) GetTargetRefByScanID(scanID int) (*snapshotdomain.ScanTargetRef, error) {
	item, err := adapter.repo.GetTargetRefByScanID(scanID)
	if err != nil {
		return nil, err
	}
	return snapshotScanTargetModelToDomain(item), nil
}
