package application

import (
	"context"
	"errors"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"

	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// TargetFacade handles target business logic.
type TargetFacade struct {
	queryService   *TargetQueryService
	commandService *TargetCommandService
}

// NewTargetFacade creates a new target service.
func NewTargetFacade(queryStore TargetQueryStore, commandStore TargetCommandStore, organizationBindingStore OrganizationTargetBindingStore) *TargetFacade {
	return &TargetFacade{
		queryService:   NewTargetQueryService(queryStore),
		commandService: NewTargetCommandService(commandStore, organizationBindingStore),
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
func (service *TargetFacade) List(query *dto.TargetListQuery) ([]Target, int64, error) {
	return service.queryService.ListTargets(context.Background(), query.GetPage(), query.GetPageSize(), query.Type, query.Filter)
}

// GetByID returns a target by ID.
func (service *TargetFacade) GetByID(id int) (*Target, error) {
	target, err := service.queryService.GetTargetByID(context.Background(), id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}
	return target, nil
}

// Update updates a target.
func (service *TargetFacade) Update(id int, req *dto.UpdateTargetRequest) (*Target, error) {
	target, err := service.commandService.UpdateTarget(context.Background(), id, req.Name)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
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
		if dberrors.IsRecordNotFound(err) {
			return ErrTargetNotFound
		}
		return err
	}
	return nil
}

// BulkDelete soft deletes multiple targets by IDs.
func (service *TargetFacade) BulkDelete(ids []int) (int64, error) {
	return service.commandService.BulkDeleteTargets(context.Background(), ids)
}
