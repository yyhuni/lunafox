package application

import (
	"context"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

func (service *ScanFacade) List(query *ScanListQuery) ([]QueryScan, int64, error) {
	page, pageSize, targetID, status, search := query.normalize()
	scans, total, err := service.queryService.ListScans(context.Background(), ScanListFilter{Page: page, PageSize: pageSize, TargetID: targetID, Status: status, Search: search})
	if err != nil {
		return nil, 0, err
	}
	return scans, total, nil
}

func (service *ScanFacade) GetByID(id int) (*QueryScan, error) {
	scan, err := service.queryService.GetScanByID(context.Background(), id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrScanNotFound
		}
		return nil, err
	}
	return scan, nil
}

func (service *ScanFacade) GetStatistics() (*ScanStatistics, error) {
	stats, err := service.queryService.GetStatistics(context.Background())
	if err != nil {
		return nil, err
	}
	return &ScanStatistics{Total: stats.Total, Running: stats.Running, Completed: stats.Completed, Failed: stats.Failed, TotalVulns: stats.TotalVulns, TotalSubdomains: stats.TotalSubdomains, TotalEndpoints: stats.TotalEndpoints, TotalWebsites: stats.TotalWebsites, TotalAssets: stats.TotalAssets}, nil
}
