package snapshotwiring

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotScreenshotQueryStoreAdapter struct {
	repo *snapshotrepo.ScreenshotSnapshotRepository
}

func newSnapshotScreenshotQueryStoreAdapter(repo *snapshotrepo.ScreenshotSnapshotRepository) *snapshotScreenshotQueryStoreAdapter {
	return &snapshotScreenshotQueryStoreAdapter{repo: repo}
}

func (adapter *snapshotScreenshotQueryStoreAdapter) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.ScreenshotSnapshot, int64, error) {
	return adapter.repo.FindByScanID(scanID, page, pageSize, filter)
}

func (adapter *snapshotScreenshotQueryStoreAdapter) FindByIDAndScanID(id int, scanID int) (*snapshotdomain.ScreenshotSnapshot, error) {
	return adapter.repo.FindByIDAndScanID(id, scanID)
}

func (adapter *snapshotScreenshotQueryStoreAdapter) CountByScanID(scanID int) (int64, error) {
	return adapter.repo.CountByScanID(scanID)
}
