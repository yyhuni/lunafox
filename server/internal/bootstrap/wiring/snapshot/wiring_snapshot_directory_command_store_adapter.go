package snapshotwiring

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotDirectoryCommandStoreAdapter struct {
	repo *snapshotrepo.DirectorySnapshotRepository
}

func newSnapshotDirectoryCommandStoreAdapter(repo *snapshotrepo.DirectorySnapshotRepository) *snapshotDirectoryCommandStoreAdapter {
	return &snapshotDirectoryCommandStoreAdapter{repo: repo}
}

func (adapter *snapshotDirectoryCommandStoreAdapter) BulkCreate(snapshots []snapshotdomain.DirectorySnapshot) (int64, error) {
	return adapter.repo.BulkCreate(snapshots)
}
