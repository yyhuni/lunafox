package snapshotwiring

import (
	"database/sql"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotWebsiteQueryStoreAdapter struct {
	repo *snapshotrepo.WebsiteSnapshotRepository
}

func newSnapshotWebsiteQueryStoreAdapter(repo *snapshotrepo.WebsiteSnapshotRepository) *snapshotWebsiteQueryStoreAdapter {
	return &snapshotWebsiteQueryStoreAdapter{repo: repo}
}

func (adapter *snapshotWebsiteQueryStoreAdapter) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.WebsiteSnapshot, int64, error) {
	return adapter.repo.FindByScanID(scanID, page, pageSize, filter)
}

func (adapter *snapshotWebsiteQueryStoreAdapter) StreamByScanID(scanID int) (*sql.Rows, error) {
	return adapter.repo.StreamByScanID(scanID)
}

func (adapter *snapshotWebsiteQueryStoreAdapter) CountByScanID(scanID int) (int64, error) {
	return adapter.repo.CountByScanID(scanID)
}

func (adapter *snapshotWebsiteQueryStoreAdapter) ScanRow(rows *sql.Rows) (*snapshotdomain.WebsiteSnapshot, error) {
	return adapter.repo.ScanRow(rows)
}
