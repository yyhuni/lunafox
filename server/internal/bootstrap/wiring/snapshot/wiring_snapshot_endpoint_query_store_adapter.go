package snapshotwiring

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotEndpointQueryStoreAdapter struct {
	repo *snapshotrepo.EndpointSnapshotRepository
}

func newSnapshotEndpointQueryStoreAdapter(repo *snapshotrepo.EndpointSnapshotRepository) *snapshotEndpointQueryStoreAdapter {
	return &snapshotEndpointQueryStoreAdapter{repo: repo}
}

func (adapter *snapshotEndpointQueryStoreAdapter) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.EndpointSnapshot, int64, error) {
	return adapter.repo.ListByScanID(scanID, page, pageSize, filter, orderBy)
}

func (adapter *snapshotEndpointQueryStoreAdapter) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	return adapter.repo.ListFilterOptionsByScanID(scanID, field)
}

func (adapter *snapshotEndpointQueryStoreAdapter) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.EndpointSnapshot) error) error {
	return adapter.repo.ForEachByScanID(ctx, scanID, visit)
}

func (adapter *snapshotEndpointQueryStoreAdapter) CountByScanID(scanID int) (int64, error) {
	return adapter.repo.CountByScanID(scanID)
}
