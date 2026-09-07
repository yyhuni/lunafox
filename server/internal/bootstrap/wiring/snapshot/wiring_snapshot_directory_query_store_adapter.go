package snapshotwiring

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotDirectoryQueryStoreAdapter struct {
	repo *snapshotrepo.DirectorySnapshotRepository
}

func newSnapshotDirectoryQueryStoreAdapter(repo *snapshotrepo.DirectorySnapshotRepository) *snapshotDirectoryQueryStoreAdapter {
	return &snapshotDirectoryQueryStoreAdapter{repo: repo}
}

func (adapter *snapshotDirectoryQueryStoreAdapter) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.DirectorySnapshot, int64, error) {
	return adapter.repo.ListByScanID(scanID, page, pageSize, filter, orderBy)
}

func (adapter *snapshotDirectoryQueryStoreAdapter) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	return adapter.repo.ListFilterOptionsByScanID(scanID, field)
}

func (adapter *snapshotDirectoryQueryStoreAdapter) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.DirectorySnapshot) error) error {
	return adapter.repo.ForEachByScanID(ctx, scanID, visit)
}

func (adapter *snapshotDirectoryQueryStoreAdapter) CountByScanID(scanID int) (int64, error) {
	return adapter.repo.CountByScanID(scanID)
}
