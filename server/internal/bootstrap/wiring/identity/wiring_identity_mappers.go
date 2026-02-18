package identitywiring

import (
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
	identitymodel "github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func identityModelUserToDomain(user *identitymodel.User) *identitydomain.User {
	if user == nil {
		return nil
	}
	return &identitydomain.User{
		ID:          user.ID,
		Password:    user.Password,
		LastLogin:   user.LastLogin,
		IsSuperuser: user.IsSuperuser,
		Username:    user.Username,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		IsStaff:     user.IsStaff,
		IsActive:    user.IsActive,
		DateJoined:  user.DateJoined,
	}
}

func identityDomainUserToModel(user *identitydomain.User) *identitymodel.User {
	if user == nil {
		return nil
	}
	return &identitymodel.User{
		ID:          user.ID,
		Password:    user.Password,
		LastLogin:   user.LastLogin,
		IsSuperuser: user.IsSuperuser,
		Username:    user.Username,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		IsStaff:     user.IsStaff,
		IsActive:    user.IsActive,
		DateJoined:  user.DateJoined,
	}
}

func identityModelOrganizationToDomain(org *identitymodel.Organization) *identitydomain.Organization {
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

func identityDomainOrganizationToModel(org *identitydomain.Organization) *identitymodel.Organization {
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

func identityModelTargetRefToDomain(target *identitymodel.OrganizationTargetRef) *identitydomain.OrganizationTargetRef {
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

func identityRepositoryOrganizationWithCountToDomain(org *identityrepo.OrganizationWithCount) *identitydomain.OrganizationWithTargetCount {
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
