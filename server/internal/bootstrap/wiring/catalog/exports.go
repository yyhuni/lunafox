package catalogwiring

import (
	"github.com/yyhuni/lunafox/server/internal/installedengines"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
	"gorm.io/gorm"
)

// NewCatalogTransactionCoordinator returns the database transaction boundary
// shared by batch target creation and organization binding.
func NewCatalogTransactionCoordinator(db *gorm.DB) catalogapp.TransactionCoordinator {
	return newCatalogTransactionCoordinator(db)
}

func NewCatalogEngineCatalogQueryStoreAdapter(query installedengines.Query) catalogapp.EngineCatalogQueryStore {
	return newEngineCatalogQueryStoreAdapter(query)
}

func NewCatalogTargetQueryStoreAdapter(repo *catalogrepo.TargetRepository) catalogapp.TargetQueryStore {
	return newCatalogTargetStoreAdapter(repo)
}

func NewCatalogTargetCommandStoreAdapter(repo *catalogrepo.TargetRepository) catalogapp.TargetCommandStore {
	return newCatalogTargetStoreAdapter(repo)
}

func NewCatalogOrganizationTargetBindingStoreAdapter(repo *identityrepo.OrganizationRepository) catalogapp.OrganizationTargetBindingStore {
	return newCatalogOrganizationTargetBindingStoreAdapter(repo)
}

func NewCatalogWordlistQueryStoreAdapter(repo *catalogrepo.WordlistRepository) catalogapp.WordlistQueryStore {
	return newCatalogWordlistStoreAdapter(repo)
}

func NewCatalogWordlistCommandStoreAdapter(repo *catalogrepo.WordlistRepository) catalogapp.WordlistCommandStore {
	return newCatalogWordlistStoreAdapter(repo)
}

func NewCatalogSubfinderProviderSettingsStoreAdapter(repo *catalogrepo.SubfinderProviderSettingsRepository) catalogapp.SubfinderAPIKeySettingsStore {
	return newCatalogSubfinderProviderSettingsStoreAdapter(repo)
}

func NewCatalogExecutionSubfinderProviderSettingsStoreAdapter(repo *catalogrepo.SubfinderProviderSettingsRepository) catalogapp.ExecutionSubfinderProviderSettingsStore {
	return newCatalogSubfinderProviderSettingsStoreAdapter(repo)
}
