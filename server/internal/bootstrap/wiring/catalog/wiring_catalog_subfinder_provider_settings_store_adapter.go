package catalogwiring

import (
	"context"

	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type catalogSubfinderProviderSettingsStoreAdapter struct {
	repo *catalogrepo.SubfinderProviderSettingsRepository
}

func newCatalogSubfinderProviderSettingsStoreAdapter(repo *catalogrepo.SubfinderProviderSettingsRepository) *catalogSubfinderProviderSettingsStoreAdapter {
	return &catalogSubfinderProviderSettingsStoreAdapter{repo: repo}
}

func (adapter *catalogSubfinderProviderSettingsStoreAdapter) GetInstance() (*catalogdomain.SubfinderProviderSettings, error) {
	settings, err := adapter.repo.GetInstance()
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, catalogapp.ErrSubfinderProviderSettingsNotFound
		}
		return nil, err
	}
	return settings, nil
}

func (adapter *catalogSubfinderProviderSettingsStoreAdapter) GetInstanceContext(ctx context.Context) (*catalogdomain.SubfinderProviderSettings, error) {
	settings, err := adapter.repo.GetInstanceContext(ctx)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, catalogapp.ErrSubfinderProviderSettingsNotFound
		}
		return nil, err
	}
	return settings, nil
}

func (adapter *catalogSubfinderProviderSettingsStoreAdapter) Update(settings *catalogdomain.SubfinderProviderSettings) error {
	return adapter.repo.Update(settings)
}
