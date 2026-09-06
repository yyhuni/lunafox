package snapshotwiring

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotScreenshotCommandStoreAdapter struct {
	repo *snapshotrepo.ScreenshotSnapshotRepository
}

func newSnapshotScreenshotCommandStoreAdapter(repo *snapshotrepo.ScreenshotSnapshotRepository) *snapshotScreenshotCommandStoreAdapter {
	return &snapshotScreenshotCommandStoreAdapter{repo: repo}
}

func (adapter *snapshotScreenshotCommandStoreAdapter) BatchUpsert(snapshots []snapshotdomain.ScreenshotSnapshot) (int64, error) {
	return adapter.repo.BatchUpsert(snapshots)
}

func (adapter *snapshotScreenshotCommandStoreAdapter) BatchUpsertContext(ctx context.Context, snapshots []snapshotdomain.ScreenshotSnapshot) (int64, error) {
	return adapter.repo.BatchUpsertContext(ctx, snapshots)
}
