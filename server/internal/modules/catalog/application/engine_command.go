package application

import (
	"context"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type EngineCommandService struct {
	store EngineCommandStore
}

func NewEngineCommandService(store EngineCommandStore) *EngineCommandService {
	return &EngineCommandService{store: store}
}

func (service *EngineCommandService) CreateEngine(ctx context.Context, name, configuration string) (*catalogdomain.ScanEngine, error) {
	_ = ctx

	engine, err := catalogdomain.NewScanEngine(name, configuration)
	if err != nil {
		return nil, err
	}

	exists, err := service.store.ExistsByName(engine.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEngineExists
	}

	if err := service.store.Create(engine); err != nil {
		return nil, err
	}

	return engine, nil
}

func (service *EngineCommandService) UpdateEngine(ctx context.Context, id int, name, configuration string) (*catalogdomain.ScanEngine, error) {
	_ = ctx

	engine, err := service.store.GetByID(id)
	if err != nil {
		return nil, err
	}

	previousName := engine.Name
	if err := engine.Rename(name); err != nil {
		return nil, err
	}
	if engine.Name != previousName {
		exists, err := service.store.ExistsByName(engine.Name, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrEngineExists
		}
	}

	engine.Reconfigure(configuration)

	if err := service.store.Update(engine); err != nil {
		return nil, err
	}

	return engine, nil
}

func (service *EngineCommandService) PatchEngine(ctx context.Context, id int, name, configuration *string) (*catalogdomain.ScanEngine, error) {
	_ = ctx

	engine, err := service.store.GetByID(id)
	if err != nil {
		return nil, err
	}

	if name != nil {
		previousName := engine.Name
		if err := engine.Rename(*name); err != nil {
			return nil, err
		}

		if engine.Name != previousName {
			exists, err := service.store.ExistsByName(engine.Name, id)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, ErrEngineExists
			}
		}
	}

	if configuration != nil {
		engine.Reconfigure(*configuration)
	}

	if err := service.store.Update(engine); err != nil {
		return nil, err
	}

	return engine, nil
}

func (service *EngineCommandService) DeleteEngine(ctx context.Context, id int) error {
	_ = ctx

	if _, err := service.store.GetByID(id); err != nil {
		return err
	}
	return service.store.Delete(id)
}
