package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type HostPortFacade struct {
	queryService *HostPortQueryService
	cmdService   *HostPortCommandService
}

func NewHostPortFacade(queryService *HostPortQueryService, cmdService *HostPortCommandService) *HostPortFacade {
	return &HostPortFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

func (service *HostPortFacade) ListByTarget(targetID int, input HostPortListQueryInput) (*HostPortListResult, error) {
	return service.ListByTargetContext(context.Background(), targetID, input)
}

// ListByTargetContext preserves caller-owned deadlines for read-only queries.
func (service *HostPortFacade) ListByTargetContext(ctx context.Context, targetID int, input HostPortListQueryInput) (*HostPortListResult, error) {
	result, err := service.queryService.ListByTarget(ctx, targetID, input)
	if err != nil {
		return nil, mapAssetTargetBoundaryError(err)
	}
	return result, nil
}

func (service *HostPortFacade) ListPortOptionsByTarget(targetID int) ([]assetdomain.FilterOption, error) {
	options, err := service.queryService.ListPortOptionsByTarget(context.Background(), targetID)
	if err != nil {
		return nil, mapAssetTargetBoundaryError(err)
	}
	return options, nil
}

func (service *HostPortFacade) ForEachByTarget(targetID int, visit func(HostPort) error) error {
	err := service.queryService.ForEachByTarget(context.Background(), targetID, visit)
	if err != nil {
		return mapAssetTargetBoundaryError(err)
	}
	return nil
}

func (service *HostPortFacade) ForEachByTargetAndIPs(targetID int, ips []string, visit func(HostPort) error) error {
	err := service.queryService.ForEachByTargetAndIPs(context.Background(), targetID, ips, visit)
	if err != nil {
		return mapAssetTargetBoundaryError(err)
	}
	return nil
}

func (service *HostPortFacade) CountByTarget(targetID int) (int64, error) {
	count, err := service.queryService.CountByTarget(context.Background(), targetID)
	if err != nil {
		return 0, mapAssetTargetBoundaryError(err)
	}
	return count, nil
}

func (service *HostPortFacade) BatchUpsert(targetID int, items []HostPortItem) (int64, error) {
	return service.BatchUpsertContext(context.Background(), targetID, items)
}

// BatchUpsertContext preserves a result-ingest caller context through asset projection.
func (service *HostPortFacade) BatchUpsertContext(ctx context.Context, targetID int, items []HostPortItem) (int64, error) {
	count, err := service.cmdService.BatchUpsert(ctx, targetID, items)
	if err != nil {
		return 0, mapAssetTargetBoundaryError(err)
	}
	return count, nil
}

func (service *HostPortFacade) BatchDeleteByIPs(ips []string) (int64, error) {
	return service.cmdService.BatchDeleteByIPs(context.Background(), ips)
}
