package domain

import "context"

// UserRepository defines persistence behaviors needed by identity use cases.
type UserRepository interface {
	GetByID(id int) (*User, error)
	FindByUsername(username string) (*User, error)
	ExistsByUsername(username string) (bool, error)
	List(page, pageSize int) ([]User, int64, error)
	Create(user *User) error
	Update(user *User) error
}

// OrganizationCommandRepository defines command-side persistence behaviors.
type OrganizationCommandRepository interface {
	GetActiveByIDContext(context.Context, int) (*Organization, error)
	ExistsByNameContext(context.Context, string, ...int) (bool, error)
	CreateContext(context.Context, *Organization) error
	UpdateContext(context.Context, *Organization) error
	SoftDeleteContext(context.Context, int) error
	BatchSoftDeleteContext(context.Context, []int) (int64, error)
	BatchAddTargetsContext(context.Context, int, []int) error
	UnlinkTargetsContext(context.Context, int, []int) (int64, error)
}

// OrganizationQueryRepository defines query-side persistence behaviors.
type OrganizationQueryRepository interface {
	GetActiveByID(id int) (*Organization, error)
	FindByIDWithCount(id int) (*OrganizationWithTargetCount, error)
	List(page, pageSize int, filter, orderBy string) ([]OrganizationWithTargetCount, int64, error)
	ListTargetsByOrganizationID(organizationID int, page, pageSize int, targetType, filter string) ([]OrganizationTargetRef, int64, error)
}
