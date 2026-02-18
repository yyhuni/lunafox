package catalogwiring

import (
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
)

type catalogTargetStoreAdapter struct {
	repo *catalogrepo.TargetRepository
}

func newCatalogTargetStoreAdapter(repo *catalogrepo.TargetRepository) *catalogTargetStoreAdapter {
	return &catalogTargetStoreAdapter{repo: repo}
}

func (adapter *catalogTargetStoreAdapter) GetActiveByID(id int) (*catalogdomain.Target, error) {
	return adapter.repo.GetActiveByID(id)
}

func (adapter *catalogTargetStoreAdapter) FindAll(page, pageSize int, targetType, filter string) ([]catalogdomain.Target, int64, error) {
	return adapter.repo.FindAll(page, pageSize, targetType, filter)
}

func (adapter *catalogTargetStoreAdapter) GetAssetCounts(targetID int) (*catalogdomain.TargetAssetCounts, error) {
	counts, err := adapter.repo.GetAssetCounts(targetID)
	if err != nil {
		return nil, err
	}
	return &catalogdomain.TargetAssetCounts{
		Subdomains:  counts.Subdomains,
		Websites:    counts.Websites,
		Endpoints:   counts.Endpoints,
		IPs:         counts.IPs,
		Directories: counts.Directories,
		Screenshots: counts.Screenshots,
	}, nil
}

func (adapter *catalogTargetStoreAdapter) GetVulnerabilityCounts(targetID int) (*catalogdomain.VulnerabilityCounts, error) {
	counts, err := adapter.repo.GetVulnerabilityCounts(targetID)
	if err != nil {
		return nil, err
	}
	return &catalogdomain.VulnerabilityCounts{
		Total:    counts.Total,
		Critical: counts.Critical,
		High:     counts.High,
		Medium:   counts.Medium,
		Low:      counts.Low,
	}, nil
}

func (adapter *catalogTargetStoreAdapter) ExistsByName(name string, excludeID ...int) (bool, error) {
	return adapter.repo.ExistsByName(name, excludeID...)
}

func (adapter *catalogTargetStoreAdapter) Create(target *catalogdomain.Target) error {
	return adapter.repo.Create(target)
}

func (adapter *catalogTargetStoreAdapter) Update(target *catalogdomain.Target) error {
	return adapter.repo.Update(target)
}

func (adapter *catalogTargetStoreAdapter) SoftDelete(id int) error {
	return adapter.repo.SoftDelete(id)
}

func (adapter *catalogTargetStoreAdapter) BulkSoftDelete(ids []int) (int64, error) {
	return adapter.repo.BulkSoftDelete(ids)
}

func (adapter *catalogTargetStoreAdapter) BulkCreateIgnoreConflicts(targets []catalogdomain.Target) (int, error) {
	return adapter.repo.BulkCreateIgnoreConflicts(targets)
}

func (adapter *catalogTargetStoreAdapter) FindByNames(names []string) ([]catalogdomain.Target, error) {
	return adapter.repo.FindByNames(names)
}
