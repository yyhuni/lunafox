package domain

// UserRepository defines persistence behaviors needed by identity use cases.
type UserRepository interface {
	GetByID(id int) (*User, error)
	FindByUsername(username string) (*User, error)
	ExistsByUsername(username string) (bool, error)
	FindAll(page, pageSize int) ([]User, int64, error)
	Create(user *User) error
	Update(user *User) error
}

// OrganizationCommandRepository defines command-side persistence behaviors.
type OrganizationCommandRepository interface {
	GetActiveByID(id int) (*Organization, error)
	ExistsByName(name string, excludeID ...int) (bool, error)
	Create(org *Organization) error
	Update(org *Organization) error
	SoftDelete(id int) error
	BulkSoftDelete(ids []int) (int64, error)
	BulkAddTargets(organizationID int, targetIDs []int) error
	UnlinkTargets(organizationID int, targetIDs []int) (int64, error)
}

// OrganizationQueryRepository defines query-side persistence behaviors.
type OrganizationQueryRepository interface {
	GetActiveByID(id int) (*Organization, error)
	FindByIDWithCount(id int) (*OrganizationWithTargetCount, error)
	FindAll(page, pageSize int, filter string) ([]OrganizationWithTargetCount, int64, error)
	FindTargets(organizationID int, page, pageSize int, targetType, filter string) ([]OrganizationTargetRef, int64, error)
}
