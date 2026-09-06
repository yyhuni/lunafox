package snapshotwiring

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotHostPortCommandStoreAdapter struct {
	repo *snapshotrepo.HostPortSnapshotRepository
}

func newSnapshotHostPortCommandStoreAdapter(repo *snapshotrepo.HostPortSnapshotRepository) *snapshotHostPortCommandStoreAdapter {
	return &snapshotHostPortCommandStoreAdapter{repo: repo}
}

func (adapter *snapshotHostPortCommandStoreAdapter) BatchCreateContext(ctx context.Context, snapshots []snapshotdomain.HostPortSnapshot) (int64, error) {
	return adapter.repo.BatchCreateContext(ctx, snapshots)
}
