package snapshotwiring

import (
	"context"

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
	return adapter.GetScanRefByIDContext(context.Background(), id)
}

func (adapter *snapshotScanRefLookupAdapter) GetScanRefByIDContext(ctx context.Context, id int) (*snapshotdomain.ScanRef, error) {
	item, err := adapter.repo.GetByIDContext(ctx, id)
	if err != nil {
		return nil, err
	}
	return snapshotScanModelToDomain(item), nil
}

func (adapter *snapshotScanRefLookupAdapter) GetTargetRefByScanID(scanID int) (*snapshotdomain.ScanTargetRef, error) {
	return adapter.GetTargetRefByScanIDContext(context.Background(), scanID)
}

func (adapter *snapshotScanRefLookupAdapter) GetTargetRefByScanIDContext(ctx context.Context, scanID int) (*snapshotdomain.ScanTargetRef, error) {
	item, err := adapter.repo.GetTargetRefByScanIDContext(ctx, scanID)
	if err != nil {
		return nil, err
	}
	return snapshotScanTargetModelToDomain(item), nil
}
