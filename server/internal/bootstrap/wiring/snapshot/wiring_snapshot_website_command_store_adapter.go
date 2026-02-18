package snapshotwiring

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotWebsiteCommandStoreAdapter struct {
	repo *snapshotrepo.WebsiteSnapshotRepository
}

func newSnapshotWebsiteCommandStoreAdapter(repo *snapshotrepo.WebsiteSnapshotRepository) *snapshotWebsiteCommandStoreAdapter {
	return &snapshotWebsiteCommandStoreAdapter{repo: repo}
}

func (adapter *snapshotWebsiteCommandStoreAdapter) BulkCreate(snapshots []snapshotdomain.WebsiteSnapshot) (int64, error) {
	return adapter.repo.BulkCreate(snapshots)
}
