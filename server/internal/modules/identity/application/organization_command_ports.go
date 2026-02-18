package application

import identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"

type OrganizationCommandStore interface {
	GetActiveByID(id int) (*identitydomain.Organization, error)
	ExistsByName(name string, excludeID ...int) (bool, error)
	Create(org *identitydomain.Organization) error
	Update(org *identitydomain.Organization) error
	SoftDelete(id int) error
	BulkSoftDelete(ids []int) (int64, error)
	BulkAddTargets(organizationID int, targetIDs []int) error
	UnlinkTargets(organizationID int, targetIDs []int) (int64, error)
}
