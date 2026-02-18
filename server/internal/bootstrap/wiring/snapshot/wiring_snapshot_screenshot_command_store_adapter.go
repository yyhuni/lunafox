package snapshotwiring

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotScreenshotCommandStoreAdapter struct {
	repo *snapshotrepo.ScreenshotSnapshotRepository
}

func newSnapshotScreenshotCommandStoreAdapter(repo *snapshotrepo.ScreenshotSnapshotRepository) *snapshotScreenshotCommandStoreAdapter {
	return &snapshotScreenshotCommandStoreAdapter{repo: repo}
}

func (adapter *snapshotScreenshotCommandStoreAdapter) BulkUpsert(snapshots []snapshotdomain.ScreenshotSnapshot) (int64, error) {
	return adapter.repo.BulkUpsert(snapshots)
}
