package snapshotwiring

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotDirectoryCommandStoreAdapter struct {
	repo *snapshotrepo.DirectorySnapshotRepository
}

func newSnapshotDirectoryCommandStoreAdapter(repo *snapshotrepo.DirectorySnapshotRepository) *snapshotDirectoryCommandStoreAdapter {
	return &snapshotDirectoryCommandStoreAdapter{repo: repo}
}

func (adapter *snapshotDirectoryCommandStoreAdapter) BatchCreate(snapshots []snapshotdomain.DirectorySnapshot) (int64, error) {
	return adapter.repo.BatchCreate(snapshots)
}

func (adapter *snapshotDirectoryCommandStoreAdapter) BatchCreateContext(ctx context.Context, snapshots []snapshotdomain.DirectorySnapshot) (int64, error) {
	return adapter.repo.BatchCreateContext(ctx, snapshots)
}
