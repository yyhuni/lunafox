package assetwiring

import (
	"database/sql"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
)

type assetHostPortStoreAdapter struct {
	repo *assetrepo.HostPortRepository
}

func newAssetHostPortStoreAdapter(repo *assetrepo.HostPortRepository) *assetHostPortStoreAdapter {
	return &assetHostPortStoreAdapter{repo: repo}
}

func (adapter *assetHostPortStoreAdapter) GetIPAggregation(targetID int, page, pageSize int, filter string) ([]assetdomain.IPAggregationRow, int64, error) {
	return adapter.repo.GetIPAggregation(targetID, page, pageSize, filter)
}

func (adapter *assetHostPortStoreAdapter) GetHostsAndPortsByIP(targetID int, ip string, filter string) ([]string, []int, error) {
	return adapter.repo.GetHostsAndPortsByIP(targetID, ip, filter)
}

func (adapter *assetHostPortStoreAdapter) StreamByTargetID(targetID int) (*sql.Rows, error) {
	return adapter.repo.StreamByTargetID(targetID)
}

func (adapter *assetHostPortStoreAdapter) StreamByTargetIDAndIPs(targetID int, ips []string) (*sql.Rows, error) {
	return adapter.repo.StreamByTargetIDAndIPs(targetID, ips)
}

func (adapter *assetHostPortStoreAdapter) CountByTargetID(targetID int) (int64, error) {
	return adapter.repo.CountByTargetID(targetID)
}

func (adapter *assetHostPortStoreAdapter) ScanRow(rows *sql.Rows) (*assetdomain.HostPort, error) {
	return adapter.repo.ScanRow(rows)
}

func (adapter *assetHostPortStoreAdapter) BulkUpsert(items []assetdomain.HostPort) (int64, error) {
	return adapter.repo.BulkUpsert(items)
}

func (adapter *assetHostPortStoreAdapter) DeleteByIPs(ips []string) (int64, error) {
	return adapter.repo.DeleteByIPs(ips)
}
