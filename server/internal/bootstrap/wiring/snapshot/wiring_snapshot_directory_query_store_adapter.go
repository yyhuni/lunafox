package snapshotwiring

import (
	"database/sql"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotDirectoryQueryStoreAdapter struct {
	repo *snapshotrepo.DirectorySnapshotRepository
}

func newSnapshotDirectoryQueryStoreAdapter(repo *snapshotrepo.DirectorySnapshotRepository) *snapshotDirectoryQueryStoreAdapter {
	return &snapshotDirectoryQueryStoreAdapter{repo: repo}
}

func (adapter *snapshotDirectoryQueryStoreAdapter) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.DirectorySnapshot, int64, error) {
	return adapter.repo.FindByScanID(scanID, page, pageSize, filter)
}

func (adapter *snapshotDirectoryQueryStoreAdapter) StreamByScanID(scanID int) (*sql.Rows, error) {
	return adapter.repo.StreamByScanID(scanID)
}

func (adapter *snapshotDirectoryQueryStoreAdapter) CountByScanID(scanID int) (int64, error) {
	return adapter.repo.CountByScanID(scanID)
}

func (adapter *snapshotDirectoryQueryStoreAdapter) ScanRow(rows *sql.Rows) (*snapshotdomain.DirectorySnapshot, error) {
	return adapter.repo.ScanRow(rows)
}
