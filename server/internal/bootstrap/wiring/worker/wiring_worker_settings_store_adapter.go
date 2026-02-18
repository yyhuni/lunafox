package workerwiring

import (
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type workerSettingsStoreAdapter struct {
	repo *catalogrepo.SubfinderProviderSettingsRepository
}

func newWorkerSettingsStoreAdapter(repo *catalogrepo.SubfinderProviderSettingsRepository) *workerSettingsStoreAdapter {
	return &workerSettingsStoreAdapter{repo: repo}
}

func (adapter *workerSettingsStoreAdapter) GetInstance() (*catalogdomain.SubfinderProviderSettings, error) {
	settings, err := adapter.repo.GetInstance()
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, catalogapp.ErrWorkerProviderConfigSettingsNotFound
		}
		return nil, err
	}
	return settings, nil
}
