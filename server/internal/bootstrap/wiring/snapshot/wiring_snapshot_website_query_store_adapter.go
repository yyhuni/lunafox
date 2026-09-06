package snapshotwiring

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotWebsiteQueryStoreAdapter struct {
	repo *snapshotrepo.WebsiteSnapshotRepository
}

func newSnapshotWebsiteQueryStoreAdapter(repo *snapshotrepo.WebsiteSnapshotRepository) *snapshotWebsiteQueryStoreAdapter {
	return &snapshotWebsiteQueryStoreAdapter{repo: repo}
}

func (adapter *snapshotWebsiteQueryStoreAdapter) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.WebsiteSnapshot, int64, error) {
	return adapter.repo.ListByScanID(scanID, page, pageSize, filter, orderBy)
}

func (adapter *snapshotWebsiteQueryStoreAdapter) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	return adapter.repo.ListFilterOptionsByScanID(scanID, field)
}

func (adapter *snapshotWebsiteQueryStoreAdapter) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.WebsiteSnapshot) error) error {
	return adapter.repo.ForEachByScanID(ctx, scanID, visit)
}

func (adapter *snapshotWebsiteQueryStoreAdapter) CountByScanID(scanID int) (int64, error) {
	return adapter.repo.CountByScanID(scanID)
}
