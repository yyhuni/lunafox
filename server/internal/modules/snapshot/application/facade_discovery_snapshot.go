package application

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

// DirectorySnapshotFacade handles directory snapshot business logic.
type DirectorySnapshotFacade struct {
	queryService *DirectorySnapshotQueryService
	cmdService   *DirectorySnapshotCommandService
}

// SubdomainSnapshotFacade handles subdomain snapshot business logic.
type SubdomainSnapshotFacade struct {
	queryService *SubdomainSnapshotQueryService
	cmdService   *SubdomainSnapshotCommandService
}

// NewDirectorySnapshotFacade creates a new directory snapshot service.
func NewDirectorySnapshotFacade(
	queryService *DirectorySnapshotQueryService,
	cmdService *DirectorySnapshotCommandService,
) *DirectorySnapshotFacade {
	return &DirectorySnapshotFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// NewSubdomainSnapshotFacade creates a new subdomain snapshot service.
func NewSubdomainSnapshotFacade(
	queryService *SubdomainSnapshotQueryService,
	cmdService *SubdomainSnapshotCommandService,
) *SubdomainSnapshotFacade {
	return &SubdomainSnapshotFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// SaveAndSync saves directory snapshots and syncs to asset table.
func (service *DirectorySnapshotFacade) SaveAndSync(scanID int, targetID int, items []DirectorySnapshotItem) (MaterializationSummary, error) {
	summary, err := service.cmdService.SaveAndSync(context.Background(), scanID, targetID, items)
	if err != nil {
		return summary, mapSnapshotSaveBoundaryError(err)
	}
	return summary, nil
}

// SaveResultBatchContext persists a fully decoded Directory result batch while
// preserving the authenticated caller context and per-item scope semantics.
func (service *DirectorySnapshotFacade) SaveResultBatchContext(ctx context.Context, scanID int, targetID int, items []DirectorySnapshotItem) (MaterializationSummary, error) {
	summary, err := service.cmdService.SaveResultBatchContext(ctx, scanID, targetID, items)
	if err != nil {
		return summary, mapSnapshotSaveBoundaryError(err)
	}
	return summary, nil
}

// ListByScan returns paginated directory snapshots for a scan.
func (service *DirectorySnapshotFacade) ListByScan(scanID int, input DirectorySnapshotListQueryInput) (*DirectorySnapshotListResult, error) {
	result, err := service.queryService.ListByScan(context.Background(), scanID, input)
	if err != nil {
		return nil, mapSnapshotScanBoundaryError(err)
	}
	return result, nil
}

func (service *DirectorySnapshotFacade) ListFilterOptionsByScan(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	options, err := service.queryService.ListFilterOptionsByScan(context.Background(), scanID, field)
	if err != nil {
		return nil, mapSnapshotScanBoundaryError(err)
	}
	return options, nil
}

// ForEachByScan streams directory snapshots for export without exposing SQL cursors.
func (service *DirectorySnapshotFacade) ForEachByScan(scanID int, visit func(DirectorySnapshot) error) error {
	err := service.queryService.ForEachByScan(context.Background(), scanID, visit)
	if err != nil {
		return mapSnapshotScanBoundaryError(err)
	}
	return nil
}

// CountByScan returns the count of directory snapshots for a scan.
func (service *DirectorySnapshotFacade) CountByScan(scanID int) (int64, error) {
	count, err := service.queryService.CountByScan(context.Background(), scanID)
	if err != nil {
		return 0, mapSnapshotScanBoundaryError(err)
	}
	return count, nil
}

// SaveAndSync saves subdomain snapshots and syncs to asset table.
func (service *SubdomainSnapshotFacade) SaveAndSync(scanID int, targetID int, items []SubdomainSnapshotItem) (MaterializationSummary, error) {
	return service.SaveAndSyncContext(context.Background(), scanID, targetID, items)
}

// SaveAndSyncContext preserves the caller context through result persistence.
func (service *SubdomainSnapshotFacade) SaveAndSyncContext(ctx context.Context, scanID int, targetID int, items []SubdomainSnapshotItem) (MaterializationSummary, error) {
	summary, err := service.cmdService.SaveAndSync(ctx, scanID, targetID, items)
	if err != nil {
		return summary, mapSubdomainSnapshotSaveBoundaryError(err)
	}
	return summary, nil
}

// ListByScan returns paginated subdomain snapshots for a scan.
func (service *SubdomainSnapshotFacade) ListByScan(scanID int, query SubdomainSnapshotListQueryInput) (*SubdomainSnapshotListResult, error) {
	result, err := service.queryService.ListByScan(context.Background(), scanID, query)
	if err != nil {
		return nil, mapSubdomainSnapshotListBoundaryError(err)
	}
	return result, nil
}

// ForEachByScan streams subdomain snapshots for export without exposing SQL cursors.
func (service *SubdomainSnapshotFacade) ForEachByScan(scanID int, visit func(SubdomainSnapshot) error) error {
	err := service.queryService.ForEachByScan(context.Background(), scanID, visit)
	if err != nil {
		return mapSnapshotScanBoundaryError(err)
	}
	return nil
}

// CountByScan returns the count of subdomain snapshots for a scan.
func (service *SubdomainSnapshotFacade) CountByScan(scanID int) (int64, error) {
	count, err := service.queryService.CountByScan(context.Background(), scanID)
	if err != nil {
		return 0, mapSnapshotScanBoundaryError(err)
	}
	return count, nil
}
