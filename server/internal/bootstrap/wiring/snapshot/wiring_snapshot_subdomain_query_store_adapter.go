package snapshotwiring

import (
	"database/sql"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotSubdomainQueryStoreAdapter struct {
	repo *snapshotrepo.SubdomainSnapshotRepository
}

func newSnapshotSubdomainQueryStoreAdapter(repo *snapshotrepo.SubdomainSnapshotRepository) *snapshotSubdomainQueryStoreAdapter {
	return &snapshotSubdomainQueryStoreAdapter{repo: repo}
}

func (adapter *snapshotSubdomainQueryStoreAdapter) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.SubdomainSnapshot, int64, error) {
	return adapter.repo.FindByScanID(scanID, page, pageSize, filter)
}

func (adapter *snapshotSubdomainQueryStoreAdapter) StreamByScanID(scanID int) (*sql.Rows, error) {
	return adapter.repo.StreamByScanID(scanID)
}

func (adapter *snapshotSubdomainQueryStoreAdapter) CountByScanID(scanID int) (int64, error) {
	return adapter.repo.CountByScanID(scanID)
}

func (adapter *snapshotSubdomainQueryStoreAdapter) ScanRow(rows *sql.Rows) (*snapshotdomain.SubdomainSnapshot, error) {
	return adapter.repo.ScanRow(rows)
}
