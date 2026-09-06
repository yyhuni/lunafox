package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type EndpointFacade struct {
	queryService *EndpointQueryService
	cmdService   *EndpointCommandService
}

func NewEndpointFacade(queryService *EndpointQueryService, cmdService *EndpointCommandService) *EndpointFacade {
	return &EndpointFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

func (service *EndpointFacade) ListByTarget(targetID int, input EndpointListQueryInput) (*EndpointListResult, error) {
	return service.ListByTargetContext(context.Background(), targetID, input)
}

// ListByTargetContext preserves caller-owned deadlines for read-only queries.
func (service *EndpointFacade) ListByTargetContext(ctx context.Context, targetID int, input EndpointListQueryInput) (*EndpointListResult, error) {
	result, err := service.queryService.ListByTarget(ctx, targetID, input)
	if err != nil {
		return nil, mapAssetTargetBoundaryError(err)
	}
	return result, nil
}

func (service *EndpointFacade) ListFilterOptionsByTarget(targetID int, field string) ([]assetdomain.FilterOption, error) {
	options, err := service.queryService.ListFilterOptionsByTarget(context.Background(), targetID, field)
	if err != nil {
		return nil, mapAssetTargetBoundaryError(err)
	}
	return options, nil
}

func (service *EndpointFacade) GetByID(id int) (*Endpoint, error) {
	item, err := service.queryService.GetByID(context.Background(), id)
	if err != nil {
		return nil, mapEndpointRecordBoundaryError(err)
	}
	return item, nil
}

func (service *EndpointFacade) BatchCreate(targetID int, urls []string) (int, error) {
	count, err := service.cmdService.BatchCreate(context.Background(), targetID, urls)
	if err != nil {
		return 0, mapAssetTargetBoundaryError(err)
	}
	return count, nil
}

func (service *EndpointFacade) Delete(id int) error {
	err := service.cmdService.Delete(context.Background(), id)
	if err != nil {
		return mapEndpointDeleteBoundaryError(err)
	}
	return nil
}

func (service *EndpointFacade) BatchDelete(ids []int) (int64, error) {
	return service.cmdService.BatchDelete(context.Background(), ids)
}

func (service *EndpointFacade) ForEachByTarget(targetID int, visit func(Endpoint) error) error {
	err := service.queryService.ForEachByTarget(context.Background(), targetID, visit)
	if err != nil {
		return mapAssetTargetBoundaryError(err)
	}
	return nil
}

func (service *EndpointFacade) CountByTarget(targetID int) (int64, error) {
	count, err := service.queryService.CountByTarget(context.Background(), targetID)
	if err != nil {
		return 0, mapAssetTargetBoundaryError(err)
	}
	return count, nil
}

func (service *EndpointFacade) BatchUpsert(targetID int, items []EndpointUpsertItem) (int64, error) {
	return service.BatchUpsertContext(context.Background(), targetID, items)
}

func (service *EndpointFacade) BatchUpsertContext(ctx context.Context, targetID int, items []EndpointUpsertItem) (int64, error) {
	affected, err := service.cmdService.BatchUpsert(ctx, targetID, items)
	if err != nil {
		return 0, mapAssetTargetBoundaryError(err)
	}
	return affected, nil
}
