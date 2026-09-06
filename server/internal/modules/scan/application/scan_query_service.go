package application

import (
	"context"
)

type ScanQueryService struct{ store ScanQueryStore }

func NewScanQueryService(store ScanQueryStore) *ScanQueryService {
	return &ScanQueryService{store: store}
}

func (service *ScanQueryService) ListScans(ctx context.Context, filter ScanListFilter) ([]QueryScan, int64, error) {
	compiledFilter := filter.Filter
	if compiledFilter == "" {
		compiledFilter = filter.Search
	}
	if err := validateScanListFilter(compiledFilter); err != nil {
		return nil, 0, err
	}
	orderBy, err := normalizeScanOrderBy(filter.OrderBy)
	if err != nil {
		return nil, 0, err
	}
	if store, ok := service.store.(ScanQueryStoreContext); ok {
		return store.ListContext(ctx, filter.Page, filter.PageSize, filter.TargetID, filter.Status, compiledFilter, orderBy)
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	return service.store.List(filter.Page, filter.PageSize, filter.TargetID, filter.Status, compiledFilter, orderBy)
}

func (service *ScanQueryService) GetScanByID(ctx context.Context, id int) (*QueryScan, error) {
	if store, ok := service.store.(ScanQueryStoreContext); ok {
		return store.GetDetailByIDContext(ctx, id)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return service.store.GetDetailByID(id)
}

func (service *ScanQueryService) GetGlobalStatsSummary(ctx context.Context) (*QueryStatistics, error) {
	_ = ctx
	return service.store.GetGlobalStatsSummary()
}
