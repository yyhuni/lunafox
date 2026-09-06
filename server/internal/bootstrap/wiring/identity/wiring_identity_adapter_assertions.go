package identitywiring

import (
	"github.com/yyhuni/lunafox/server/internal/auth"
	identityapp "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
)

var _ identityapp.UserQueryStore = (*identityUserStoreAdapter)(nil)
var _ identityapp.UserCommandStore = (*identityUserStoreAdapter)(nil)
var _ identityapp.AuthUserStore = (*identityUserStoreAdapter)(nil)
var _ auth.TokenVersionReader = (*identityUserStoreAdapter)(nil)
var _ identityapp.AdminPasswordResetStore = (*identityUserStoreAdapter)(nil)
var _ identityapp.OrganizationQueryStore = (*identityOrganizationStoreAdapter)(nil)
var _ identityapp.OrganizationCommandStore = (*identityOrganizationStoreAdapter)(nil)
