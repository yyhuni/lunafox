package catalogwiring

import "context"

import identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"

type catalogOrganizationTargetBindingStoreAdapter struct {
	repo *identityrepo.OrganizationRepository
}

func newCatalogOrganizationTargetBindingStoreAdapter(repo *identityrepo.OrganizationRepository) *catalogOrganizationTargetBindingStoreAdapter {
	return &catalogOrganizationTargetBindingStoreAdapter{repo: repo}
}

func (adapter *catalogOrganizationTargetBindingStoreAdapter) ExistsByID(id int) (bool, error) {
	return adapter.repo.Exists(id)
}

func (adapter *catalogOrganizationTargetBindingStoreAdapter) ExistsByIDContext(ctx context.Context, id int) (bool, error) {
	return adapter.repo.ExistsContext(ctx, id)
}

func (adapter *catalogOrganizationTargetBindingStoreAdapter) BatchAddTargets(organizationID int, targetIDs []int) error {
	return adapter.repo.BatchAddTargets(organizationID, targetIDs)
}

func (adapter *catalogOrganizationTargetBindingStoreAdapter) BatchAddTargetsContext(ctx context.Context, organizationID int, targetIDs []int) error {
	return adapter.repo.BatchAddTargetsContext(ctx, organizationID, targetIDs)
}
