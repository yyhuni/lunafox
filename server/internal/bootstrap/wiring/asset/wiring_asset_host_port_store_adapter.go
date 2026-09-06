package assetwiring

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
)

type assetHostPortStoreAdapter struct {
	repo *assetrepo.HostPortRepository
}

func newAssetHostPortStoreAdapter(repo *assetrepo.HostPortRepository) *assetHostPortStoreAdapter {
	return &assetHostPortStoreAdapter{repo: repo}
}

func (adapter *assetHostPortStoreAdapter) GetIPAggregation(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.IPAggregationRow, int64, error) {
	return adapter.repo.GetIPAggregation(targetID, page, pageSize, filter, orderBy)
}

func (adapter *assetHostPortStoreAdapter) GetIPAggregationContext(ctx context.Context, targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.IPAggregationRow, int64, error) {
	return adapter.repo.GetIPAggregationContext(ctx, targetID, page, pageSize, filter, orderBy)
}

func (adapter *assetHostPortStoreAdapter) GetHostsAndPortsByIP(targetID int, ip string, filter string) ([]string, []int, error) {
	return adapter.repo.GetHostsAndPortsByIP(targetID, ip, filter)
}

func (adapter *assetHostPortStoreAdapter) GetHostsAndPortsByIPContext(ctx context.Context, targetID int, ip string, filter string) ([]string, []int, error) {
	return adapter.repo.GetHostsAndPortsByIPContext(ctx, targetID, ip, filter)
}

func (adapter *assetHostPortStoreAdapter) ListPortOptionsByTargetID(targetID int) ([]assetdomain.FilterOption, error) {
	return adapter.repo.ListPortOptionsByTargetID(targetID)
}

func (adapter *assetHostPortStoreAdapter) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.HostPort) error) error {
	return adapter.repo.ForEachByTargetID(ctx, targetID, visit)
}

func (adapter *assetHostPortStoreAdapter) ForEachByTargetIDAndIPs(ctx context.Context, targetID int, ips []string, visit func(assetdomain.HostPort) error) error {
	return adapter.repo.ForEachByTargetIDAndIPs(ctx, targetID, ips, visit)
}

func (adapter *assetHostPortStoreAdapter) CountByTargetID(targetID int) (int64, error) {
	return adapter.repo.CountByTargetID(targetID)
}

func (adapter *assetHostPortStoreAdapter) BatchUpsert(items []assetdomain.HostPort) (int64, error) {
	return adapter.repo.BatchUpsert(items)
}

func (adapter *assetHostPortStoreAdapter) BatchUpsertContext(ctx context.Context, items []assetdomain.HostPort) (int64, error) {
	return adapter.repo.BatchUpsertContext(ctx, items)
}

func (adapter *assetHostPortStoreAdapter) DeleteByIPs(ips []string) (int64, error) {
	return adapter.repo.DeleteByIPs(ips)
}
