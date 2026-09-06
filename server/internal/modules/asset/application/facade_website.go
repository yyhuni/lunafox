package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type WebsiteFacade struct {
	queryService *WebsiteQueryService
	cmdService   *WebsiteCommandService
}

func NewWebsiteFacade(queryService *WebsiteQueryService, cmdService *WebsiteCommandService) *WebsiteFacade {
	return &WebsiteFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

func (service *WebsiteFacade) ListByTarget(targetID int, input WebsiteListQueryInput) (*WebsiteListResult, error) {
	return service.ListByTargetContext(context.Background(), targetID, input)
}

// ListByTargetContext preserves caller-owned deadlines for read-only queries.
func (service *WebsiteFacade) ListByTargetContext(ctx context.Context, targetID int, input WebsiteListQueryInput) (*WebsiteListResult, error) {
	result, err := service.queryService.ListByTarget(ctx, targetID, input)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Get returns one Website read model by its resource ID.
func (service *WebsiteFacade) Get(id int) (*WebsiteReadModel, error) {
	website, err := service.queryService.Get(context.Background(), id)
	if err != nil {
		return nil, mapWebsiteRecordBoundaryError(err)
	}
	return website, nil
}

func (service *WebsiteFacade) ListFilterOptionsByTarget(targetID int, field string) ([]assetdomain.FilterOption, error) {
	options, err := service.queryService.ListFilterOptionsByTarget(context.Background(), targetID, field)
	if err != nil {
		return nil, mapAssetTargetBoundaryError(err)
	}
	return options, nil
}

func (service *WebsiteFacade) BatchCreate(targetID int, urls []string) (int, error) {
	return service.BatchCreateContext(context.Background(), targetID, urls)
}

// BatchCreateContext preserves a result-ingest caller context through asset projection.
func (service *WebsiteFacade) BatchCreateContext(ctx context.Context, targetID int, urls []string) (int, error) {
	count, err := service.cmdService.BatchCreate(ctx, targetID, urls)
	if err != nil {
		return 0, mapAssetTargetDomainError(err)
	}
	return count, nil
}

func (service *WebsiteFacade) Delete(id int) error {
	err := service.cmdService.Delete(context.Background(), id)
	if err != nil {
		return mapWebsiteRecordBoundaryError(err)
	}
	return nil
}

func (service *WebsiteFacade) BatchDelete(ids []int) (int64, error) {
	return service.cmdService.BatchDelete(context.Background(), ids)
}

func (service *WebsiteFacade) ForEachByTarget(targetID int, visit func(Website) error) error {
	err := service.queryService.ForEachByTarget(context.Background(), targetID, visit)
	if err != nil {
		return mapAssetTargetBoundaryError(err)
	}
	return nil
}

func (service *WebsiteFacade) CountByTarget(targetID int) (int64, error) {
	count, err := service.queryService.CountByTarget(context.Background(), targetID)
	if err != nil {
		return 0, mapAssetTargetBoundaryError(err)
	}
	return count, nil
}

func (service *WebsiteFacade) BatchUpsert(targetID int, items []WebsiteUpsertItem) (int64, error) {
	return service.BatchUpsertContext(context.Background(), targetID, items)
}

// BatchUpsertContext preserves a result-ingest caller context through asset projection.
func (service *WebsiteFacade) BatchUpsertContext(ctx context.Context, targetID int, items []WebsiteUpsertItem) (int64, error) {
	count, err := service.cmdService.BatchUpsert(ctx, targetID, items)
	if err != nil {
		return 0, mapAssetTargetDomainError(err)
	}
	return count, nil
}

// BatchUpsertTechnologyContext is the dedicated current-only result path.
func (service *WebsiteFacade) BatchUpsertTechnologyContext(ctx context.Context, targetID int, items []assetdomain.WebsiteTechnology) (WebsiteTechnologyMaterializationSummary, error) {
	summary, err := service.cmdService.BatchUpsertTechnologyContext(ctx, targetID, items)
	if err != nil {
		return summary, mapAssetTargetDomainError(err)
	}
	return summary, nil
}
