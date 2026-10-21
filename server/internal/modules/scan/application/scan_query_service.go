package application

import (
	"context"
)

type ScanQueryService struct {
	store                ScanQueryStore
	agentExecutionReader ScanAgentExecutionReader
}

func NewScanQueryService(store ScanQueryStore, agentExecutionReader ScanAgentExecutionReader) *ScanQueryService {
	return &ScanQueryService{store: store, agentExecutionReader: agentExecutionReader}
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
	var scan *QueryScan
	var err error
	if store, ok := service.store.(ScanQueryStoreContext); ok {
		scan, err = store.GetDetailByIDContext(ctx, id)
	} else {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		scan, err = service.store.GetDetailByID(id)
	}
	if err != nil || scan == nil {
		return scan, err
	}
	// This is a current read projection, never a persisted failure or a rebind.
	// Running work remains valid; only unclaimed executable work is waiting.
	scan.ExecutionBlockedReason = nil
	if scan.AgentID == nil || scan.AgentDeleted || !hasUnclaimedExecutableTask(scan) {
		return scan, nil
	}
	eligibility, err := readAgentExecutionEligibility(ctx, service.agentExecutionReader, *scan.AgentID)
	if err != nil {
		return nil, err
	}
	if eligibility == agentExecutionLicenseRestricted {
		reason := ExecutionBlockedReasonAgentLicenseRestricted
		scan.ExecutionBlockedReason = &reason
	}
	return scan, nil
}

func hasUnclaimedExecutableTask(scan *QueryScan) bool {
	if scan.Status != "pending" && scan.Status != "running" {
		return false
	}
	for _, task := range scan.RuntimeTasks {
		if task.HasResolvedExecutionPlan && (task.Status == "pending" || task.Status == "blocked") {
			return true
		}
	}
	return false
}

func (service *ScanQueryService) GetGlobalStatsSummary(ctx context.Context) (*QueryStatistics, error) {
	_ = ctx
	return service.store.GetGlobalStatsSummary()
}
