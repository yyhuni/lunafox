package snapshotwiring

import (
	"database/sql"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotEndpointQueryStoreAdapter struct {
	repo *snapshotrepo.EndpointSnapshotRepository
}

func newSnapshotEndpointQueryStoreAdapter(repo *snapshotrepo.EndpointSnapshotRepository) *snapshotEndpointQueryStoreAdapter {
	return &snapshotEndpointQueryStoreAdapter{repo: repo}
}

func (adapter *snapshotEndpointQueryStoreAdapter) FindByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.EndpointSnapshot, int64, error) {
	return adapter.repo.FindByScanID(scanID, page, pageSize, filter)
}

func (adapter *snapshotEndpointQueryStoreAdapter) StreamByScanID(scanID int) (*sql.Rows, error) {
	return adapter.repo.StreamByScanID(scanID)
}

func (adapter *snapshotEndpointQueryStoreAdapter) CountByScanID(scanID int) (int64, error) {
	return adapter.repo.CountByScanID(scanID)
}

func (adapter *snapshotEndpointQueryStoreAdapter) ScanRow(rows *sql.Rows) (*snapshotdomain.EndpointSnapshot, error) {
	return adapter.repo.ScanRow(rows)
}
