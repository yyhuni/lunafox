package identitywiring

import identityapp "github.com/yyhuni/lunafox/server/internal/modules/identity/application"

var _ identityapp.UserQueryStore = (*identityUserStoreAdapter)(nil)
var _ identityapp.UserCommandStore = (*identityUserStoreAdapter)(nil)
var _ identityapp.AuthUserStore = (*identityUserStoreAdapter)(nil)
var _ identityapp.OrganizationQueryStore = (*identityOrganizationStoreAdapter)(nil)
var _ identityapp.OrganizationCommandStore = (*identityOrganizationStoreAdapter)(nil)
