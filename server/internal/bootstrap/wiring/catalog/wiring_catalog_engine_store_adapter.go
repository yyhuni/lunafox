package catalogwiring

import (
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
)

type catalogEngineStoreAdapter struct {
	repo *catalogrepo.EngineRepository
}

func newCatalogEngineStoreAdapter(repo *catalogrepo.EngineRepository) *catalogEngineStoreAdapter {
	return &catalogEngineStoreAdapter{repo: repo}
}

func (adapter *catalogEngineStoreAdapter) GetByID(id int) (*catalogdomain.ScanEngine, error) {
	return adapter.repo.GetByID(id)
}

func (adapter *catalogEngineStoreAdapter) FindAll(page, pageSize int) ([]catalogdomain.ScanEngine, int64, error) {
	return adapter.repo.FindAll(page, pageSize)
}

func (adapter *catalogEngineStoreAdapter) ExistsByName(name string, excludeID ...int) (bool, error) {
	return adapter.repo.ExistsByName(name, excludeID...)
}

func (adapter *catalogEngineStoreAdapter) Create(engine *catalogdomain.ScanEngine) error {
	return adapter.repo.Create(engine)
}

func (adapter *catalogEngineStoreAdapter) Update(engine *catalogdomain.ScanEngine) error {
	return adapter.repo.Update(engine)
}

func (adapter *catalogEngineStoreAdapter) Delete(id int) error {
	return adapter.repo.Delete(id)
}
