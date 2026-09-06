package application

import (
	"context"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

type OrganizationCommandStore interface {
	GetActiveByIDContext(context.Context, int) (*identitydomain.Organization, error)
	ExistsByNameContext(context.Context, string, ...int) (bool, error)
	CreateContext(context.Context, *identitydomain.Organization) error
	UpdateContext(context.Context, *identitydomain.Organization) error
	SoftDeleteContext(context.Context, int) error
	BatchSoftDeleteContext(context.Context, []int) (int64, error)
	BatchAddTargetsContext(context.Context, int, []int) error
	UnlinkTargetsContext(context.Context, int, []int) (int64, error)
}
