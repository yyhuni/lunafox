package identitywiring

import (
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
	identitymodel "github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func identityModelUserToIdentityDomainUser(user *identitymodel.User) *identitydomain.User {
	if user == nil {
		return nil
	}
	return &identitydomain.User{
		ID:                  user.ID,
		Password:            user.Password,
		TokenVersion:        user.TokenVersion,
		LastLogin:           user.LastLogin,
		IsSuperuser:         user.IsSuperuser,
		Username:            user.Username,
		FirstName:           user.FirstName,
		LastName:            user.LastName,
		Email:               user.Email,
		Locale:              user.Locale,
		LocaleInitializedAt: user.LocaleInitializedAt,
		IsStaff:             user.IsStaff,
		IsActive:            user.IsActive,
		DateJoined:          user.DateJoined,
	}
}

func identityModelUsersToIdentityDomainUsers(users []identitymodel.User) []identitydomain.User {
	results := make([]identitydomain.User, 0, len(users))
	for index := range users {
		results = append(results, *identityModelUserToIdentityDomainUser(&users[index]))
	}
	return results
}

func identityDomainUserToIdentityModelUser(user *identitydomain.User) *identitymodel.User {
	if user == nil {
		return nil
	}
	return &identitymodel.User{
		ID:                  user.ID,
		Password:            user.Password,
		TokenVersion:        user.TokenVersion,
		LastLogin:           user.LastLogin,
		IsSuperuser:         user.IsSuperuser,
		Username:            user.Username,
		FirstName:           user.FirstName,
		LastName:            user.LastName,
		Email:               user.Email,
		Locale:              user.Locale,
		LocaleInitializedAt: user.LocaleInitializedAt,
		IsStaff:             user.IsStaff,
		IsActive:            user.IsActive,
		DateJoined:          user.DateJoined,
	}
}

func identityModelOrganizationToIdentityDomainOrganization(org *identitymodel.Organization) *identitydomain.Organization {
	if org == nil {
		return nil
	}
	return &identitydomain.Organization{
		ID:          org.ID,
		Name:        org.Name,
		Description: org.Description,
		CreatedAt:   timeutil.ToUTC(org.CreatedAt),
		DeletedAt:   timeutil.ToUTCPtr(org.DeletedAt),
	}
}

func identityDomainOrganizationToIdentityModelOrganization(org *identitydomain.Organization) *identitymodel.Organization {
	if org == nil {
		return nil
	}
	return &identitymodel.Organization{
		ID:          org.ID,
		Name:        org.Name,
		Description: org.Description,
		CreatedAt:   timeutil.ToUTC(org.CreatedAt),
		DeletedAt:   timeutil.ToUTCPtr(org.DeletedAt),
	}
}

func identityModelTargetRefToIdentityDomainTargetRef(target *identitymodel.OrganizationTargetRef) *identitydomain.OrganizationTargetRef {
	if target == nil {
		return nil
	}
	return &identitydomain.OrganizationTargetRef{
		ID:            target.ID,
		Name:          target.Name,
		Type:          target.Type,
		CreatedAt:     timeutil.ToUTC(target.CreatedAt),
		LastScannedAt: timeutil.ToUTCPtr(target.LastScannedAt),
		DeletedAt:     timeutil.ToUTCPtr(target.DeletedAt),
	}
}

func identityModelTargetRefsToIdentityDomainTargetRefs(targets []identitymodel.OrganizationTargetRef) []identitydomain.OrganizationTargetRef {
	results := make([]identitydomain.OrganizationTargetRef, 0, len(targets))
	for index := range targets {
		results = append(results, *identityModelTargetRefToIdentityDomainTargetRef(&targets[index]))
	}
	return results
}

func identityRepositoryOrganizationWithCountToIdentityDomainOrganizationWithTargetCount(org *identityrepo.OrganizationWithCount) *identitydomain.OrganizationWithTargetCount {
	if org == nil {
		return nil
	}
	return &identitydomain.OrganizationWithTargetCount{
		Organization: identitydomain.Organization{
			ID:          org.ID,
			Name:        org.Name,
			Description: org.Description,
			CreatedAt:   timeutil.ToUTC(org.CreatedAt),
			DeletedAt:   timeutil.ToUTCPtr(org.DeletedAt),
		},
		TargetCount: org.TargetCount,
	}
}

func identityRepositoryOrganizationWithCountsToIdentityDomainOrganizationWithTargetCounts(orgs []identityrepo.OrganizationWithCount) []identitydomain.OrganizationWithTargetCount {
	results := make([]identitydomain.OrganizationWithTargetCount, 0, len(orgs))
	for index := range orgs {
		results = append(results, *identityRepositoryOrganizationWithCountToIdentityDomainOrganizationWithTargetCount(&orgs[index]))
	}
	return results
}
