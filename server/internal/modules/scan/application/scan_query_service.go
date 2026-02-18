package application

import (
	"context"
)

type ScanQueryService struct{ store ScanQueryStore }

func NewScanQueryService(store ScanQueryStore) *ScanQueryService {
	return &ScanQueryService{store: store}
}

func (service *ScanQueryService) ListScans(ctx context.Context, filter ScanListFilter) ([]QueryScan, int64, error) {
	_ = ctx
	return service.store.FindAll(filter.Page, filter.PageSize, filter.TargetID, filter.Status, filter.Search)
}

func (service *ScanQueryService) GetScanByID(ctx context.Context, id int) (*QueryScan, error) {
	_ = ctx
	return service.store.GetQueryByID(id)
}

func (service *ScanQueryService) GetStatistics(ctx context.Context) (*QueryStatistics, error) {
	_ = ctx
	return service.store.GetStatistics()
}
