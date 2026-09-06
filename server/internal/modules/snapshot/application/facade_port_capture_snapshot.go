package application

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

// HostPortSnapshotFacade handles host-port snapshot business logic.
type HostPortSnapshotFacade struct {
	queryService *HostPortSnapshotQueryService
	cmdService   *HostPortSnapshotCommandService
}

// ScreenshotSnapshotFacade handles screenshot snapshot business logic.
type ScreenshotSnapshotFacade struct {
	queryService *ScreenshotSnapshotQueryService
	cmdService   *ScreenshotSnapshotCommandService
}

// NewHostPortSnapshotFacade creates a new host-port snapshot service.
func NewHostPortSnapshotFacade(
	queryService *HostPortSnapshotQueryService,
	cmdService *HostPortSnapshotCommandService,
) *HostPortSnapshotFacade {
	return &HostPortSnapshotFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// NewScreenshotSnapshotFacade creates a new screenshot snapshot service.
func NewScreenshotSnapshotFacade(
	queryService *ScreenshotSnapshotQueryService,
	cmdService *ScreenshotSnapshotCommandService,
) *ScreenshotSnapshotFacade {
	return &ScreenshotSnapshotFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// SaveAndSync saves host-port snapshots and syncs to asset table.
func (service *HostPortSnapshotFacade) SaveAndSync(scanID int, targetID int, items []HostPortSnapshotItem) (MaterializationSummary, error) {
	return service.SaveAndSyncContext(context.Background(), scanID, targetID, items)
}

// SaveAndSyncContext preserves the caller context through result persistence.
func (service *HostPortSnapshotFacade) SaveAndSyncContext(ctx context.Context, scanID int, targetID int, items []HostPortSnapshotItem) (MaterializationSummary, error) {
	summary, err := service.cmdService.SaveAndSync(ctx, scanID, targetID, items)
	if err != nil {
		return summary, mapSnapshotSaveBoundaryError(err)
	}
	return summary, nil
}

// ListByScan returns paginated host-port snapshot IP aggregates for a scan.
func (service *HostPortSnapshotFacade) ListByScan(scanID int, query HostPortSnapshotListQueryInput) (*HostPortSnapshotListResult, error) {
	result, err := service.queryService.ListByScan(context.Background(), scanID, query)
	if err != nil {
		return nil, mapSnapshotScanBoundaryError(err)
	}
	return result, nil
}

func (service *HostPortSnapshotFacade) ListPortOptionsByScan(scanID int) ([]snapshotdomain.FilterOption, error) {
	options, err := service.queryService.ListPortOptionsByScan(context.Background(), scanID)
	if err != nil {
		return nil, mapSnapshotScanBoundaryError(err)
	}
	return options, nil
}

// ForEachByScan streams host-port snapshots for export without exposing SQL cursors.
func (service *HostPortSnapshotFacade) ForEachByScan(scanID int, visit func(HostPortSnapshot) error) error {
	err := service.queryService.ForEachByScan(context.Background(), scanID, visit)
	if err != nil {
		return mapSnapshotScanBoundaryError(err)
	}
	return nil
}

// CountByScan returns the count of host-port snapshots for a scan.
func (service *HostPortSnapshotFacade) CountByScan(scanID int) (int64, error) {
	count, err := service.queryService.CountByScan(context.Background(), scanID)
	if err != nil {
		return 0, mapSnapshotScanBoundaryError(err)
	}
	return count, nil
}

// SaveAndSync saves screenshot snapshots and syncs to asset table.
func (service *ScreenshotSnapshotFacade) SaveAndSync(scanID int, targetID int, items []ScreenshotSnapshotItem) (MaterializationSummary, error) {
	return service.SaveAndSyncContext(context.Background(), scanID, targetID, items)
}

// SaveAndSyncContext preserves the caller context through result persistence.
func (service *ScreenshotSnapshotFacade) SaveAndSyncContext(ctx context.Context, scanID int, targetID int, items []ScreenshotSnapshotItem) (MaterializationSummary, error) {
	summary, err := service.cmdService.SaveAndSync(ctx, scanID, targetID, items)
	if err != nil {
		return summary, mapSnapshotSaveBoundaryError(err)
	}
	return summary, nil
}

// SaveResultBatchContext keeps screenshot writes inside the outer result-ingest
// transaction after its Task lease and Target state have been revalidated.
func (service *ScreenshotSnapshotFacade) SaveResultBatchContext(ctx context.Context, scanID int, targetID int, items []ScreenshotSnapshotItem) (MaterializationSummary, error) {
	summary, err := service.cmdService.SaveResultBatchContext(ctx, scanID, targetID, items)
	if err != nil {
		return summary, mapSnapshotSaveBoundaryError(err)
	}
	return summary, nil
}

// ListByScan returns paginated screenshot snapshots for a scan.
func (service *ScreenshotSnapshotFacade) ListByScan(scanID int, query ScreenshotSnapshotListQueryInput) (*ScreenshotSnapshotListResult, error) {
	result, err := service.queryService.ListByScan(context.Background(), scanID, query)
	if err != nil {
		return nil, mapSnapshotScanBoundaryError(err)
	}
	return result, nil
}

func (service *ScreenshotSnapshotFacade) ListFilterOptionsByScan(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	options, err := service.queryService.ListFilterOptionsByScan(context.Background(), scanID, field)
	if err != nil {
		return nil, mapSnapshotScanBoundaryError(err)
	}
	return options, nil
}

// GetByID returns a screenshot snapshot by ID under a scan (including image data).
func (service *ScreenshotSnapshotFacade) GetByID(scanID int, id int) (*ScreenshotSnapshot, error) {
	item, err := service.queryService.GetByID(context.Background(), scanID, id)
	if err != nil {
		return nil, mapScreenshotSnapshotBoundaryError(err)
	}
	return item, nil
}
