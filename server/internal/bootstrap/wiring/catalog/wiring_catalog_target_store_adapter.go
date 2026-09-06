package catalogwiring

import (
	"context"

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

func (adapter *catalogTargetStoreAdapter) List(page, pageSize int, filter, orderBy string) ([]catalogdomain.Target, int64, error) {
	return adapter.repo.List(page, pageSize, filter, orderBy)
}

func (adapter *catalogTargetStoreAdapter) ListContext(ctx context.Context, page, pageSize int, filter, orderBy string) ([]catalogdomain.Target, int64, error) {
	return adapter.repo.ListContext(ctx, page, pageSize, filter, orderBy)
}

func (adapter *catalogTargetStoreAdapter) GetActiveByIDContext(ctx context.Context, id int) (*catalogdomain.Target, error) {
	return adapter.repo.GetActiveByIDContext(ctx, id)
}

func (adapter *catalogTargetStoreAdapter) GetAssetCountsSummary(targetID int) (*catalogdomain.TargetAssetCounts, error) {
	counts, err := adapter.repo.GetAssetCountsSummary(targetID)
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

func (adapter *catalogTargetStoreAdapter) GetAssetCountsSummaryContext(ctx context.Context, targetID int) (*catalogdomain.TargetAssetCounts, error) {
	counts, err := adapter.repo.GetAssetCountsSummaryContext(ctx, targetID)
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

func (adapter *catalogTargetStoreAdapter) GetVulnerabilityCountsSummary(targetID int) (*catalogdomain.VulnerabilityCounts, error) {
	counts, err := adapter.repo.GetVulnerabilityCountsSummary(targetID)
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

func (adapter *catalogTargetStoreAdapter) GetVulnerabilityCountsSummaryContext(ctx context.Context, targetID int) (*catalogdomain.VulnerabilityCounts, error) {
	counts, err := adapter.repo.GetVulnerabilityCountsSummaryContext(ctx, targetID)
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

func (adapter *catalogTargetStoreAdapter) BatchSoftDelete(ids []int) (int64, error) {
	return adapter.repo.BatchSoftDelete(ids)
}

func (adapter *catalogTargetStoreAdapter) TombstoneAndEnsureCleanup(ctx context.Context, id int) (bool, error) {
	return adapter.repo.TombstoneAndEnsureCleanup(ctx, id)
}

func (adapter *catalogTargetStoreAdapter) BatchTombstoneAndEnsureCleanup(ctx context.Context, ids []int) (int64, error) {
	return adapter.repo.BatchTombstoneAndEnsureCleanup(ctx, ids)
}

func (adapter *catalogTargetStoreAdapter) BatchCreateIgnoreConflicts(targets []catalogdomain.Target) (int, error) {
	return adapter.repo.BatchCreateIgnoreConflicts(targets)
}

func (adapter *catalogTargetStoreAdapter) BatchCreateIgnoreConflictsContext(ctx context.Context, targets []catalogdomain.Target) (int, error) {
	return adapter.repo.BatchCreateIgnoreConflictsContext(ctx, targets)
}

func (adapter *catalogTargetStoreAdapter) FindByNames(names []string) ([]catalogdomain.Target, error) {
	return adapter.repo.FindByNames(names)
}

func (adapter *catalogTargetStoreAdapter) FindByNamesContext(ctx context.Context, names []string) ([]catalogdomain.Target, error) {
	return adapter.repo.FindByNamesContext(ctx, names)
}
