package application

type OrganizationTargetBindingStore interface {
	ExistsByID(id int) (bool, error)
	BulkAddTargets(organizationID int, targetIDs []int) error
}
