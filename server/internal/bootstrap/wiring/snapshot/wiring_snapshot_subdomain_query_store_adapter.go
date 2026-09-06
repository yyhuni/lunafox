package snapshotwiring

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotSubdomainQueryStoreAdapter struct {
	repo *snapshotrepo.SubdomainSnapshotRepository
}

func newSnapshotSubdomainQueryStoreAdapter(repo *snapshotrepo.SubdomainSnapshotRepository) *snapshotSubdomainQueryStoreAdapter {
	return &snapshotSubdomainQueryStoreAdapter{repo: repo}
}

func (adapter *snapshotSubdomainQueryStoreAdapter) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.SubdomainSnapshot, int64, error) {
	return adapter.repo.ListByScanID(scanID, page, pageSize, filter, orderBy)
}

func (adapter *snapshotSubdomainQueryStoreAdapter) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.SubdomainSnapshot) error) error {
	return adapter.repo.ForEachByScanID(ctx, scanID, visit)
}

func (adapter *snapshotSubdomainQueryStoreAdapter) CountByScanID(scanID int) (int64, error) {
	return adapter.repo.CountByScanID(scanID)
}
