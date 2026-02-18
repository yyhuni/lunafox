package catalogwiring

import (
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
)

type catalogWordlistStoreAdapter struct {
	repo *catalogrepo.WordlistRepository
}

func newCatalogWordlistStoreAdapter(repo *catalogrepo.WordlistRepository) *catalogWordlistStoreAdapter {
	return &catalogWordlistStoreAdapter{repo: repo}
}

func (adapter *catalogWordlistStoreAdapter) FindAll(page, pageSize int) ([]catalogdomain.Wordlist, int64, error) {
	return adapter.repo.FindAll(page, pageSize)
}

func (adapter *catalogWordlistStoreAdapter) List() ([]catalogdomain.Wordlist, error) {
	return adapter.repo.List()
}

func (adapter *catalogWordlistStoreAdapter) GetByID(id int) (*catalogdomain.Wordlist, error) {
	return adapter.repo.GetByID(id)
}

func (adapter *catalogWordlistStoreAdapter) FindByName(name string) (*catalogdomain.Wordlist, error) {
	return adapter.repo.FindByName(name)
}

func (adapter *catalogWordlistStoreAdapter) ExistsByName(name string) (bool, error) {
	return adapter.repo.ExistsByName(name)
}

func (adapter *catalogWordlistStoreAdapter) Create(wordlist *catalogdomain.Wordlist) error {
	return adapter.repo.Create(wordlist)
}

func (adapter *catalogWordlistStoreAdapter) Update(wordlist *catalogdomain.Wordlist) error {
	return adapter.repo.Update(wordlist)
}

func (adapter *catalogWordlistStoreAdapter) Delete(id int) error {
	return adapter.repo.Delete(id)
}
