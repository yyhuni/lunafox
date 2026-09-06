package catalogwiring

import (
	"context"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
)

type catalogWordlistStoreAdapter struct {
	repo *catalogrepo.WordlistRepository
}

func newCatalogWordlistStoreAdapter(repo *catalogrepo.WordlistRepository) *catalogWordlistStoreAdapter {
	return &catalogWordlistStoreAdapter{repo: repo}
}

func (adapter *catalogWordlistStoreAdapter) List(page, pageSize int, filter, orderBy string) ([]catalogdomain.Wordlist, int64, error) {
	return adapter.repo.List(page, pageSize, filter, orderBy)
}

func (adapter *catalogWordlistStoreAdapter) ListContext(ctx context.Context, page, pageSize int, filter, orderBy string) ([]catalogdomain.Wordlist, int64, error) {
	return adapter.repo.ListContext(ctx, page, pageSize, filter, orderBy)
}

func (adapter *catalogWordlistStoreAdapter) ListAll() ([]catalogdomain.Wordlist, error) {
	return adapter.repo.ListAll()
}

func (adapter *catalogWordlistStoreAdapter) ListAllContext(ctx context.Context) ([]catalogdomain.Wordlist, error) {
	return adapter.repo.ListAllContext(ctx)
}

func (adapter *catalogWordlistStoreAdapter) ListTagSummaries(page, pageSize int, filter string) ([]catalogdomain.WordlistTagSummary, int64, error) {
	return adapter.repo.ListTagSummaries(page, pageSize, filter)
}

func (adapter *catalogWordlistStoreAdapter) GetByID(id int) (*catalogdomain.Wordlist, error) {
	return adapter.repo.GetByID(id)
}

func (adapter *catalogWordlistStoreAdapter) GetByIDContext(ctx context.Context, id int) (*catalogdomain.Wordlist, error) {
	return adapter.repo.GetByIDContext(ctx, id)
}

func (adapter *catalogWordlistStoreAdapter) ExistsByFileName(fileName string, excludeID ...int) (bool, error) {
	return adapter.repo.ExistsByFileName(fileName, excludeID...)
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
