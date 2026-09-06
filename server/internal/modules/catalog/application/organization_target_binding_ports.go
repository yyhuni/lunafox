package application

import "context"

type OrganizationTargetBindingStore interface {
	ExistsByID(id int) (bool, error)
	BatchAddTargets(organizationID int, targetIDs []int) error
}

// OrganizationTargetBindingContextStore participates in the shared catalog
// transaction and preserves the active request context.
type OrganizationTargetBindingContextStore interface {
	ExistsByIDContext(context.Context, int) (bool, error)
	BatchAddTargetsContext(context.Context, int, []int) error
}
