package snapshotwiring

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotSubdomainCommandStoreAdapter struct {
	repo *snapshotrepo.SubdomainSnapshotRepository
}

func newSnapshotSubdomainCommandStoreAdapter(repo *snapshotrepo.SubdomainSnapshotRepository) *snapshotSubdomainCommandStoreAdapter {
	return &snapshotSubdomainCommandStoreAdapter{repo: repo}
}

func (adapter *snapshotSubdomainCommandStoreAdapter) BatchCreateContext(ctx context.Context, snapshots []snapshotdomain.SubdomainSnapshot) (int64, error) {
	return adapter.repo.BatchCreateContext(ctx, snapshots)
}
