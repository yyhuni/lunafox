package application

import (
	"context"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type EngineQueryService struct {
	store EngineQueryStore
}

func NewEngineQueryService(store EngineQueryStore) *EngineQueryService {
	return &EngineQueryService{store: store}
}

func (service *EngineQueryService) ListEngines(ctx context.Context, page, pageSize int) ([]catalogdomain.ScanEngine, int64, error) {
	_ = ctx
	return service.store.FindAll(page, pageSize)
}

func (service *EngineQueryService) GetEngineByID(ctx context.Context, id int) (*catalogdomain.ScanEngine, error) {
	_ = ctx
	return service.store.GetByID(id)
}
