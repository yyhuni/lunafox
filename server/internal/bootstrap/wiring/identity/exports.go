package identitywiring

import (
	"github.com/yyhuni/lunafox/server/internal/auth"
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

// NewIdentityTokenVersionReaderAdapter exposes the current user token version.
func NewIdentityTokenVersionReaderAdapter(repo *identityrepo.UserRepository) auth.TokenVersionReader {
	return newIdentityUserStoreAdapter(repo)
}

// NewIdentityAdminPasswordResetStoreAdapter exposes the reset transaction boundary.
func NewIdentityAdminPasswordResetStoreAdapter(repo *identityrepo.UserRepository) identityapp.AdminPasswordResetStore {
	return newIdentityUserStoreAdapter(repo)
}

func NewIdentityOrganizationQueryStoreAdapter(repo *identityrepo.OrganizationRepository) identityapp.OrganizationQueryStore {
	return newIdentityOrganizationStoreAdapter(repo)
}

func NewIdentityOrganizationCommandStoreAdapter(repo *identityrepo.OrganizationRepository) identityapp.OrganizationCommandStore {
	return newIdentityOrganizationStoreAdapter(repo)
}
