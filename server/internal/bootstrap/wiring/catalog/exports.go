package catalogwiring

import (
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
)

func NewCatalogEngineQueryStoreAdapter(repo *catalogrepo.EngineRepository) catalogapp.EngineQueryStore {
	return newCatalogEngineStoreAdapter(repo)
}

func NewCatalogEngineCommandStoreAdapter(repo *catalogrepo.EngineRepository) catalogapp.EngineCommandStore {
	return newCatalogEngineStoreAdapter(repo)
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
