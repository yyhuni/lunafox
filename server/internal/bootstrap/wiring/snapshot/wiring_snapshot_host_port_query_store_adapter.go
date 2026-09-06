package snapshotwiring

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotHostPortQueryStoreAdapter struct {
	repo *snapshotrepo.HostPortSnapshotRepository
}

func newSnapshotHostPortQueryStoreAdapter(repo *snapshotrepo.HostPortSnapshotRepository) *snapshotHostPortQueryStoreAdapter {
	return &snapshotHostPortQueryStoreAdapter{repo: repo}
}

func (adapter *snapshotHostPortQueryStoreAdapter) GetIPAggregation(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.HostPortIPAggregationRow, int64, error) {
	return adapter.repo.GetIPAggregation(scanID, page, pageSize, filter, orderBy)
}

func (adapter *snapshotHostPortQueryStoreAdapter) GetHostsAndPortsByIP(scanID int, ip string, filter string) ([]string, []int, error) {
	return adapter.repo.GetHostsAndPortsByIP(scanID, ip, filter)
}

func (adapter *snapshotHostPortQueryStoreAdapter) ListPortOptionsByScanID(scanID int) ([]snapshotdomain.FilterOption, error) {
	return adapter.repo.ListPortOptionsByScanID(scanID)
}

func (adapter *snapshotHostPortQueryStoreAdapter) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.HostPortSnapshot) error) error {
	return adapter.repo.ForEachByScanID(ctx, scanID, visit)
}

func (adapter *snapshotHostPortQueryStoreAdapter) CountByScanID(scanID int) (int64, error) {
	return adapter.repo.CountByScanID(scanID)
}
