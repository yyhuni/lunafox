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

func (adapter *snapshotScreenshotQueryStoreAdapter) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.ScreenshotSnapshot, int64, error) {
	return adapter.repo.ListByScanID(scanID, page, pageSize, filter, orderBy)
}

func (adapter *snapshotScreenshotQueryStoreAdapter) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	return adapter.repo.ListFilterOptionsByScanID(scanID, field)
}

func (adapter *snapshotScreenshotQueryStoreAdapter) FindByIDAndScanID(id int, scanID int) (*snapshotdomain.ScreenshotSnapshot, error) {
	return adapter.repo.FindByIDAndScanID(id, scanID)
}

func (adapter *snapshotScreenshotQueryStoreAdapter) CountByScanID(scanID int) (int64, error) {
	return adapter.repo.CountByScanID(scanID)
}
