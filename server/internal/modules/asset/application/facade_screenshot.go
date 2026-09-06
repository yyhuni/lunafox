package application

import (
	"context"
	"fmt"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type ScreenshotFacade struct {
	queryService *ScreenshotQueryService
	cmdService   *ScreenshotCommandService
}

func NewScreenshotFacade(queryService *ScreenshotQueryService, cmdService *ScreenshotCommandService) *ScreenshotFacade {
	return &ScreenshotFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

func (service *ScreenshotFacade) ListByTarget(targetID int, input ScreenshotListQueryInput) (*ScreenshotListResult, error) {
	return service.ListByTargetContext(context.Background(), targetID, input)
}

// ListByTargetContext preserves the caller-owned cancellation and deadline.
func (service *ScreenshotFacade) ListByTargetContext(ctx context.Context, targetID int, input ScreenshotListQueryInput) (*ScreenshotListResult, error) {
	if service == nil || service.queryService == nil {
		return nil, fmt.Errorf("screenshot query service is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	result, err := service.queryService.ListByTarget(ctx, targetID, input)
	if err != nil {
		return nil, mapAssetTargetBoundaryError(err)
	}
	return result, nil
}

func (service *ScreenshotFacade) ListFilterOptionsByTarget(targetID int, field string) ([]assetdomain.FilterOption, error) {
	options, err := service.queryService.ListFilterOptionsByTarget(context.Background(), targetID, field)
	if err != nil {
		return nil, mapAssetTargetBoundaryError(err)
	}
	return options, nil
}

func (service *ScreenshotFacade) GetByID(id int) (*Screenshot, error) {
	return service.GetByIDContext(context.Background(), id)
}

func (service *ScreenshotFacade) GetByIDContext(ctx context.Context, id int) (*Screenshot, error) {
	if service == nil || service.queryService == nil {
		return nil, fmt.Errorf("screenshot query service is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	item, err := service.queryService.GetByID(ctx, id)
	if err != nil {
		return nil, mapScreenshotRecordBoundaryError(err)
	}
	return item, nil
}

// GetByIDForTargetContext validates ownership before returning image bytes.
// This keeps target-scoped MCP image reads from becoming an ID-only escape hatch.
func (service *ScreenshotFacade) GetByIDForTargetContext(ctx context.Context, targetID, id int) (*Screenshot, error) {
	if service == nil || service.queryService == nil {
		return nil, fmt.Errorf("screenshot query service is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	item, err := service.queryService.GetByID(ctx, id)
	if err != nil {
		return nil, mapScreenshotRecordBoundaryError(err)
	}
	if item == nil || item.TargetID != targetID {
		return nil, ErrScreenshotNotFound
	}
	return item, nil
}

func (service *ScreenshotFacade) BatchDelete(ids []int) (int64, error) {
	return service.cmdService.BatchDelete(context.Background(), ids)
}

func (service *ScreenshotFacade) BatchUpsert(targetID int, req *BatchUpsertScreenshotRequest) (int64, error) {
	return service.BatchUpsertContext(context.Background(), targetID, req)
}

func (service *ScreenshotFacade) BatchUpsertContext(ctx context.Context, targetID int, req *BatchUpsertScreenshotRequest) (int64, error) {
	count, err := service.cmdService.BatchUpsert(ctx, targetID, req)
	if err != nil {
		return 0, mapAssetTargetBoundaryError(err)
	}
	return count, nil
}
