package application

import (
	"context"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type TargetSummary struct {
	Subdomains      int64
	Websites        int64
	Endpoints       int64
	IPs             int64
	Directories     int64
	Screenshots     int64
	Vulnerabilities *VulnerabilitySummary
}

type VulnerabilitySummary struct {
	Total    int64
	Critical int64
	High     int64
	Medium   int64
	Low      int64
}

type TargetQueryService struct {
	store TargetQueryStore
}

func NewTargetQueryService(store TargetQueryStore) *TargetQueryService {
	return &TargetQueryService{store: store}
}

func (service *TargetQueryService) ListTargets(ctx context.Context, page, pageSize int, targetType, filter string) ([]catalogdomain.Target, int64, error) {
	_ = ctx
	return service.store.FindAll(page, pageSize, targetType, filter)
}

func (service *TargetQueryService) GetTargetByID(ctx context.Context, id int) (*catalogdomain.Target, error) {
	_ = ctx
	return service.store.GetActiveByID(id)
}

func (service *TargetQueryService) GetTargetDetailByID(ctx context.Context, id int) (*catalogdomain.Target, *TargetSummary, error) {
	_ = ctx

	target, err := service.store.GetActiveByID(id)
	if err != nil {
		return nil, nil, err
	}

	assetCounts, err := service.store.GetAssetCounts(id)
	if err != nil {
		return nil, nil, err
	}
	vulnCounts, err := service.store.GetVulnerabilityCounts(id)
	if err != nil {
		return nil, nil, err
	}

	summary := &TargetSummary{
		Subdomains:  assetCounts.Subdomains,
		Websites:    assetCounts.Websites,
		Endpoints:   assetCounts.Endpoints,
		IPs:         assetCounts.IPs,
		Directories: assetCounts.Directories,
		Screenshots: assetCounts.Screenshots,
		Vulnerabilities: &VulnerabilitySummary{
			Total:    vulnCounts.Total,
			Critical: vulnCounts.Critical,
			High:     vulnCounts.High,
			Medium:   vulnCounts.Medium,
			Low:      vulnCounts.Low,
		},
	}

	return target, summary, nil
}
