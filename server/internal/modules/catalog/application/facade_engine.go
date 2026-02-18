package application

import (
	"context"
	"errors"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"

	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// EngineFacade handles engine business logic.
type EngineFacade struct {
	queryService *EngineQueryService
	cmdService   *EngineCommandService
}

// NewEngineFacade creates a new engine service.
func NewEngineFacade(queryStore EngineQueryStore, commandStore EngineCommandStore) *EngineFacade {
	return &EngineFacade{
		queryService: NewEngineQueryService(queryStore),
		cmdService:   NewEngineCommandService(commandStore),
	}
}

// Create creates a new engine.
func (service *EngineFacade) Create(req *dto.CreateEngineRequest) (*ScanEngine, error) {
	engine, err := service.cmdService.CreateEngine(context.Background(), req.Name, req.Configuration)
	if err != nil {
		if errors.Is(err, ErrEngineExists) {
			return nil, ErrEngineExists
		}
		if errors.Is(err, ErrInvalidEngine) {
			return nil, ErrInvalidEngine
		}
		return nil, err
	}
	return engine, nil
}

// List returns paginated engines.
func (service *EngineFacade) List(query *dto.PaginationQuery) ([]ScanEngine, int64, error) {
	return service.queryService.ListEngines(context.Background(), query.GetPage(), query.GetPageSize())
}

// GetByID returns an engine by ID.
func (service *EngineFacade) GetByID(id int) (*ScanEngine, error) {
	engine, err := service.queryService.GetEngineByID(context.Background(), id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrEngineNotFound
		}
		return nil, err
	}
	return engine, nil
}

// Update updates an engine.
func (service *EngineFacade) Update(id int, req *dto.UpdateEngineRequest) (*ScanEngine, error) {
	engine, err := service.cmdService.UpdateEngine(context.Background(), id, req.Name, req.Configuration)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrEngineNotFound
		}
		if errors.Is(err, ErrEngineExists) {
			return nil, ErrEngineExists
		}
		if errors.Is(err, ErrInvalidEngine) {
			return nil, ErrInvalidEngine
		}
		return nil, err
	}
	return engine, nil
}

// Patch partially updates an engine.
func (service *EngineFacade) Patch(id int, req *dto.PatchEngineRequest) (*ScanEngine, error) {
	engine, err := service.cmdService.PatchEngine(context.Background(), id, req.Name, req.Configuration)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrEngineNotFound
		}
		if errors.Is(err, ErrEngineExists) {
			return nil, ErrEngineExists
		}
		if errors.Is(err, ErrInvalidEngine) {
			return nil, ErrInvalidEngine
		}
		return nil, err
	}
	return engine, nil
}

// Delete deletes an engine.
func (service *EngineFacade) Delete(id int) error {
	err := service.cmdService.DeleteEngine(context.Background(), id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrEngineNotFound
		}
		return err
	}
	return nil
}
