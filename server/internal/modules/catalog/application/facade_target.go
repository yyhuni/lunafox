package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// TargetFacade handles target business logic.
type TargetFacade struct {
	queryService   *TargetQueryService
	commandService *TargetCommandService
}

// NewTargetFacade creates a new target service.
func NewTargetFacade(queryService *TargetQueryService, commandService *TargetCommandService) *TargetFacade {
	return &TargetFacade{
		queryService:   queryService,
		commandService: commandService,
	}
}

// Create creates a new target.
func (service *TargetFacade) Create(req *dto.CreateTargetRequest) (*Target, error) {
	target, err := service.commandService.CreateTarget(context.Background(), req.Name)
	if err != nil {
		if errors.Is(err, ErrTargetExists) {
			return nil, ErrTargetExists
		}
		if errors.Is(err, ErrInvalidTarget) {
			return nil, ErrInvalidTarget
		}
		return nil, err
	}
	return target, nil
}

// List returns paginated targets.
func (service *TargetFacade) List(query *dto.TargetListQuery) (*TargetListResult, error) {
	return service.ListContext(context.Background(), TargetListQueryInput{
		PageSize:  query.GetPageSize(),
		PageToken: query.PageToken,
		Filter:    query.Filter,
		OrderBy:   query.OrderBy,
	})
}

// ListContext preserves the caller-owned cancellation and deadline for read-only callers.
func (service *TargetFacade) ListContext(ctx context.Context, input TargetListQueryInput) (*TargetListResult, error) {
	if service == nil || service.queryService == nil {
		return nil, fmt.Errorf("target query service is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	return service.queryService.ListTargets(ctx, input)
}

// GetByID returns a target by ID.
func (service *TargetFacade) GetByID(id int) (*Target, error) {
	target, err := service.GetByIDContext(context.Background(), id)
	return target, err
}

// GetByIDContext preserves the caller-owned cancellation and deadline.
func (service *TargetFacade) GetByIDContext(ctx context.Context, id int) (*Target, error) {
	if service == nil || service.queryService == nil {
		return nil, fmt.Errorf("target query service is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	target, err := service.queryService.GetTargetByID(ctx, id)
	if err != nil {
		return nil, mapTargetBoundaryError(err)
	}
	return target, nil
}

// GetDetailContext returns the target and persisted aggregate summary.
func (service *TargetFacade) GetDetailContext(ctx context.Context, id int) (*Target, *TargetSummary, error) {
	if service == nil || service.queryService == nil {
		return nil, nil, fmt.Errorf("target query service is not configured")
	}
	if ctx == nil {
		return nil, nil, context.Canceled
	}
	target, summary, err := service.queryService.GetTargetDetailByID(ctx, id)
	if err != nil {
		return nil, nil, mapTargetBoundaryError(err)
	}
	return target, summary, nil
}

// Update updates a target.
func (service *TargetFacade) Update(id int, req *dto.UpdateTargetRequest) (*Target, error) {
	target, err := service.commandService.UpdateTarget(context.Background(), id, req.DisplayName)
	if err != nil {
		if isRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		if errors.Is(err, ErrTargetExists) {
			return nil, ErrTargetExists
		}
		if errors.Is(err, ErrInvalidTarget) {
			return nil, ErrInvalidTarget
		}
		return nil, err
	}
	return target, nil
}

// Delete soft deletes a target.
func (service *TargetFacade) Delete(id int) error {
	err := service.commandService.DeleteTarget(context.Background(), id)
	if err != nil {
		return mapTargetBoundaryError(err)
	}
	return nil
}

// BatchDelete soft deletes multiple targets by IDs.
func (service *TargetFacade) BatchDelete(ids []int) (int64, error) {
	deletedCount, err := service.commandService.BatchDeleteTargets(context.Background(), ids)
	if err != nil {
		return 0, mapTargetBoundaryError(err)
	}
	return deletedCount, nil
}
