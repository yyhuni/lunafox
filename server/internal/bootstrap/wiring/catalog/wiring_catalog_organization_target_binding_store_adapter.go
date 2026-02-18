package catalogwiring

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

func (adapter *catalogOrganizationTargetBindingStoreAdapter) BulkAddTargets(organizationID int, targetIDs []int) error {
	return adapter.repo.BulkAddTargets(organizationID, targetIDs)
}
