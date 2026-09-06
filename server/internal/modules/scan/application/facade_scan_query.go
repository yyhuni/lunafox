package application

import (
	"context"
)

func (service *ScanFacade) List(query *ScanListQuery) ([]QueryScan, int64, error) {
	page, pageSize, targetID, status, search, filter, orderBy := query.normalize()
	return service.ListContext(context.Background(), ScanListFilter{Page: page, PageSize: pageSize, TargetID: targetID, Status: status, Search: search, Filter: filter, OrderBy: orderBy})
}

// ListContext preserves a caller-owned deadline for scan list queries.
func (service *ScanFacade) ListContext(ctx context.Context, filter ScanListFilter) ([]QueryScan, int64, error) {
	scans, total, err := service.queryService.ListScans(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return scans, total, nil
}

func (service *ScanFacade) GetByID(id int) (*QueryScan, error) {
	return service.GetByIDContext(context.Background(), id)
}

// GetByIDContext preserves a caller-owned deadline for scan detail queries.
func (service *ScanFacade) GetByIDContext(ctx context.Context, id int) (*QueryScan, error) {
	scan, err := service.queryService.GetScanByID(ctx, id)
	if err != nil {
		return nil, mapScanBoundaryError(err)
	}
	return scan, nil
}

func (service *ScanFacade) GetGlobalStatsSummary() (*ScanStatistics, error) {
	stats, err := service.queryService.GetGlobalStatsSummary(context.Background())
	if err != nil {
		return nil, err
	}
	return &ScanStatistics{Total: stats.Total, Pending: stats.Pending, Running: stats.Running, Completed: stats.Completed, Failed: stats.Failed, Cancelled: stats.Cancelled, TotalVulns: stats.TotalVulns, TotalSubdomains: stats.TotalSubdomains, TotalEndpoints: stats.TotalEndpoints, TotalWebsites: stats.TotalWebsites, TotalAssets: stats.TotalAssets}, nil
}
