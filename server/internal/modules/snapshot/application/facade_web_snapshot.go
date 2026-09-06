package application

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

// WebsiteSnapshotFacade handles website snapshot business logic.
type WebsiteSnapshotFacade struct {
	queryService *WebsiteSnapshotQueryService
	cmdService   *WebsiteSnapshotCommandService
}

// EndpointSnapshotFacade handles endpoint snapshot business logic.
type EndpointSnapshotFacade struct {
	queryService *EndpointSnapshotQueryService
	cmdService   *EndpointSnapshotCommandService
}

// NewWebsiteSnapshotFacade creates a new website snapshot service.
func NewWebsiteSnapshotFacade(
	queryService *WebsiteSnapshotQueryService,
	cmdService *WebsiteSnapshotCommandService,
) *WebsiteSnapshotFacade {
	return &WebsiteSnapshotFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// NewEndpointSnapshotFacade creates a new endpoint snapshot service.
func NewEndpointSnapshotFacade(
	queryService *EndpointSnapshotQueryService,
	cmdService *EndpointSnapshotCommandService,
) *EndpointSnapshotFacade {
	return &EndpointSnapshotFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// SaveAndSync saves website snapshots and syncs to asset table.
func (service *WebsiteSnapshotFacade) SaveAndSync(scanID int, targetID int, items []WebsiteSnapshotItem) (MaterializationSummary, error) {
	return service.SaveAndSyncContext(context.Background(), scanID, targetID, items)
}

// SaveAndSyncContext preserves the caller context through result persistence.
func (service *WebsiteSnapshotFacade) SaveAndSyncContext(ctx context.Context, scanID int, targetID int, items []WebsiteSnapshotItem) (MaterializationSummary, error) {
	summary, err := service.cmdService.SaveAndSync(ctx, scanID, targetID, items)
	if err != nil {
		return summary, mapSnapshotSaveBoundaryError(err)
	}
	return summary, nil
}

// ListByScan returns paginated website snapshots for a scan.
func (service *WebsiteSnapshotFacade) ListByScan(scanID int, input WebsiteSnapshotListQueryInput) (*WebsiteSnapshotListResult, error) {
	result, err := service.queryService.ListByScan(context.Background(), scanID, input)
	if err != nil {
		return nil, mapSnapshotScanBoundaryError(err)
	}
	return result, nil
}

func (service *WebsiteSnapshotFacade) ListFilterOptionsByScan(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	options, err := service.queryService.ListFilterOptionsByScan(context.Background(), scanID, field)
	if err != nil {
		return nil, mapSnapshotScanBoundaryError(err)
	}
	return options, nil
}

// ForEachByScan streams website snapshots for export without exposing SQL cursors.
func (service *WebsiteSnapshotFacade) ForEachByScan(scanID int, visit func(WebsiteSnapshot) error) error {
	err := service.queryService.ForEachByScan(context.Background(), scanID, visit)
	if err != nil {
		return mapSnapshotScanBoundaryError(err)
	}
	return nil
}

// CountByScan returns the count of website snapshots for a scan.
func (service *WebsiteSnapshotFacade) CountByScan(scanID int) (int64, error) {
	count, err := service.queryService.CountByScan(context.Background(), scanID)
	if err != nil {
		return 0, mapSnapshotScanBoundaryError(err)
	}
	return count, nil
}

// SaveAndSync saves endpoint snapshots and syncs to asset table.
func (service *EndpointSnapshotFacade) SaveAndSync(scanID int, targetID int, items []EndpointSnapshotItem) (MaterializationSummary, error) {
	return service.SaveAndSyncContext(context.Background(), scanID, targetID, items)
}

func (service *EndpointSnapshotFacade) SaveAndSyncContext(ctx context.Context, scanID int, targetID int, items []EndpointSnapshotItem) (MaterializationSummary, error) {
	summary, err := service.cmdService.SaveAndSync(ctx, scanID, targetID, items)
	if err != nil {
		return summary, mapSnapshotSaveBoundaryError(err)
	}
	return summary, nil
}

// ListByScan returns paginated endpoint snapshots for a scan.
func (service *EndpointSnapshotFacade) ListByScan(scanID int, input EndpointSnapshotListQueryInput) (*EndpointSnapshotListResult, error) {
	result, err := service.queryService.ListByScan(context.Background(), scanID, input)
	if err != nil {
		return nil, mapSnapshotScanBoundaryError(err)
	}
	return result, nil
}

func (service *EndpointSnapshotFacade) ListFilterOptionsByScan(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	options, err := service.queryService.ListFilterOptionsByScan(context.Background(), scanID, field)
	if err != nil {
		return nil, mapSnapshotScanBoundaryError(err)
	}
	return options, nil
}

// ForEachByScan streams endpoint snapshots for export without exposing SQL cursors.
func (service *EndpointSnapshotFacade) ForEachByScan(scanID int, visit func(EndpointSnapshot) error) error {
	err := service.queryService.ForEachByScan(context.Background(), scanID, visit)
	if err != nil {
		return mapSnapshotScanBoundaryError(err)
	}
	return nil
}

// CountByScan returns the count of endpoint snapshots for a scan.
func (service *EndpointSnapshotFacade) CountByScan(scanID int) (int64, error) {
	count, err := service.queryService.CountByScan(context.Background(), scanID)
	if err != nil {
		return 0, mapSnapshotScanBoundaryError(err)
	}
	return count, nil
}
