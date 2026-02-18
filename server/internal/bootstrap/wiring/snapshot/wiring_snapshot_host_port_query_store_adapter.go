package snapshotwiring

import (
	"database/sql"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotHostPortQueryStoreAdapter struct {
	repo *snapshotrepo.HostPortSnapshotRepository
}

func newSnapshotHostPortQueryStoreAdapter(repo *snapshotrepo.HostPortSnapshotRepository) *snapshotHostPortQueryStoreAdapter {
	return &snapshotHostPortQueryStoreAdapter{repo: repo}
}

func (adapter *snapshotHostPortQueryStoreAdapter) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.HostPortSnapshot, int64, error) {
	return adapter.repo.FindByScanID(scanID, page, pageSize, filter)
}

func (adapter *snapshotHostPortQueryStoreAdapter) StreamByScanID(scanID int) (*sql.Rows, error) {
	return adapter.repo.StreamByScanID(scanID)
}

func (adapter *snapshotHostPortQueryStoreAdapter) CountByScanID(scanID int) (int64, error) {
	return adapter.repo.CountByScanID(scanID)
}

func (adapter *snapshotHostPortQueryStoreAdapter) ScanRow(rows *sql.Rows) (*snapshotdomain.HostPortSnapshot, error) {
	return adapter.repo.ScanRow(rows)
}
