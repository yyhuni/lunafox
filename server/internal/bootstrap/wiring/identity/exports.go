package identitywiring

import (
	identityapp "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
)

func NewIdentityUserQueryStoreAdapter(repo *identityrepo.UserRepository) identityapp.UserQueryStore {
	return newIdentityUserStoreAdapter(repo)
}

func NewIdentityUserCommandStoreAdapter(repo *identityrepo.UserRepository) identityapp.UserCommandStore {
	return newIdentityUserStoreAdapter(repo)
}

func NewIdentityAuthUserStoreAdapter(repo *identityrepo.UserRepository) identityapp.AuthUserStore {
	return newIdentityUserStoreAdapter(repo)
}

func NewIdentityOrganizationQueryStoreAdapter(repo *identityrepo.OrganizationRepository) identityapp.OrganizationQueryStore {
	return newIdentityOrganizationStoreAdapter(repo)
}

func NewIdentityOrganizationCommandStoreAdapter(repo *identityrepo.OrganizationRepository) identityapp.OrganizationCommandStore {
	return newIdentityOrganizationStoreAdapter(repo)
}
