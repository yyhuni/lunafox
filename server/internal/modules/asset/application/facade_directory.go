package application

import (
	"context"
	"errors"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

var ErrDirectoryNotFound = errors.New("directory not found")

type DirectoryFacade struct {
	queryService *DirectoryQueryService
	cmdService   *DirectoryCommandService
}

func NewDirectoryFacade(queryService *DirectoryQueryService, cmdService *DirectoryCommandService) *DirectoryFacade {
	return &DirectoryFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

func (service *DirectoryFacade) ListByTarget(targetID int, input DirectoryListQueryInput) (*DirectoryListResult, error) {
	return service.ListByTargetContext(context.Background(), targetID, input)
}

// ListByTargetContext preserves caller-owned deadlines for read-only queries.
func (service *DirectoryFacade) ListByTargetContext(ctx context.Context, targetID int, input DirectoryListQueryInput) (*DirectoryListResult, error) {
	result, err := service.queryService.ListByTarget(ctx, targetID, input)
	if err != nil {
		return nil, mapAssetTargetBoundaryError(err)
	}
	return result, nil
}

func (service *DirectoryFacade) ListFilterOptionsByTarget(targetID int, field string) ([]assetdomain.FilterOption, error) {
	options, err := service.queryService.ListFilterOptionsByTarget(context.Background(), targetID, field)
	if err != nil {
		return nil, mapAssetTargetBoundaryError(err)
	}
	return options, nil
}

func (service *DirectoryFacade) BatchCreate(targetID int, urls []string) (int, error) {
	count, err := service.cmdService.BatchCreate(context.Background(), targetID, urls)
	if err != nil {
		return 0, mapAssetTargetBoundaryError(err)
	}
	return count, nil
}

func (service *DirectoryFacade) BatchDelete(ids []int) (int64, error) {
	return service.cmdService.BatchDelete(context.Background(), ids)
}

func (service *DirectoryFacade) ForEachByTarget(targetID int, visit func(Directory) error) error {
	err := service.queryService.ForEachByTarget(context.Background(), targetID, visit)
	if err != nil {
		return mapAssetTargetBoundaryError(err)
	}
	return nil
}

func (service *DirectoryFacade) CountByTarget(targetID int) (int64, error) {
	count, err := service.queryService.CountByTarget(context.Background(), targetID)
	if err != nil {
		return 0, mapAssetTargetBoundaryError(err)
	}
	return count, nil
}

func (service *DirectoryFacade) BatchUpsert(targetID int, items []DirectoryUpsertItem) (int64, error) {
	return service.BatchUpsertContext(context.Background(), targetID, items)
}

func (service *DirectoryFacade) BatchUpsertContext(ctx context.Context, targetID int, items []DirectoryUpsertItem) (int64, error) {
	affected, err := service.cmdService.BatchUpsert(ctx, targetID, items)
	if err != nil {
		return 0, mapAssetTargetBoundaryError(err)
	}
	return affected, nil
}
