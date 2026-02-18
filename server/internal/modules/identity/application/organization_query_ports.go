package application

import identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"

type OrganizationQueryStore interface {
	GetActiveByID(id int) (*identitydomain.Organization, error)
	FindByIDWithCount(id int) (*identitydomain.OrganizationWithTargetCount, error)
	FindAll(page, pageSize int, filter string) ([]identitydomain.OrganizationWithTargetCount, int64, error)
	FindTargets(organizationID int, page, pageSize int, targetType, filter string) ([]identitydomain.OrganizationTargetRef, int64, error)
}
