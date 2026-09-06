package application

import (
	"context"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

type OrganizationQueryStore interface {
	GetActiveByID(id int) (*identitydomain.Organization, error)
	FindByIDWithCount(id int) (*identitydomain.OrganizationWithTargetCount, error)
	List(page, pageSize int, filter, orderBy string) ([]identitydomain.OrganizationWithTargetCount, int64, error)
	ListTargetsByOrganizationID(organizationID int, page, pageSize int, targetType, filter string) ([]identitydomain.OrganizationTargetRef, int64, error)
}

// OrganizationQueryStoreContext is an optional context-aware extension. The
// legacy query port remains intact so existing module test doubles and
// non-request callers do not need to grow a context parameter all at once.
type OrganizationQueryStoreContext interface {
	GetActiveByIDContext(context.Context, int) (*identitydomain.Organization, error)
	FindByIDWithCountContext(context.Context, int) (*identitydomain.OrganizationWithTargetCount, error)
	ListContext(context.Context, int, int, string, string) ([]identitydomain.OrganizationWithTargetCount, int64, error)
	ListTargetsByOrganizationIDContext(context.Context, int, int, int, string, string) ([]identitydomain.OrganizationTargetRef, int64, error)
}
