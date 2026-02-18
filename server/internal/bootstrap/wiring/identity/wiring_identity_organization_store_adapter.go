package identitywiring

import (
	"errors"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
)

type identityOrganizationStoreAdapter struct {
	repo *identityrepo.OrganizationRepository
}

func newIdentityOrganizationStoreAdapter(repo *identityrepo.OrganizationRepository) *identityOrganizationStoreAdapter {
	return &identityOrganizationStoreAdapter{repo: repo}
}

func (adapter *identityOrganizationStoreAdapter) GetActiveByID(id int) (*identitydomain.Organization, error) {
	org, err := adapter.repo.GetActiveByID(id)
	if err != nil {
		return nil, err
	}
	return identityModelOrganizationToDomain(org), nil
}

func (adapter *identityOrganizationStoreAdapter) FindByIDWithCount(id int) (*identitydomain.OrganizationWithTargetCount, error) {
	org, err := adapter.repo.FindByIDWithCount(id)
	if err != nil {
		return nil, err
	}
	return identityRepositoryOrganizationWithCountToDomain(org), nil
}

func (adapter *identityOrganizationStoreAdapter) FindAll(page, pageSize int, filter string) ([]identitydomain.OrganizationWithTargetCount, int64, error) {
	orgs, total, err := adapter.repo.FindAll(page, pageSize, filter)
	if err != nil {
		return nil, 0, err
	}

	results := make([]identitydomain.OrganizationWithTargetCount, 0, len(orgs))
	for index := range orgs {
		results = append(results, *identityRepositoryOrganizationWithCountToDomain(&orgs[index]))
	}
	return results, total, nil
}

func (adapter *identityOrganizationStoreAdapter) FindTargets(organizationID int, page, pageSize int, targetType, filter string) ([]identitydomain.OrganizationTargetRef, int64, error) {
	targets, total, err := adapter.repo.FindTargets(organizationID, page, pageSize, targetType, filter)
	if err != nil {
		return nil, 0, err
	}

	results := make([]identitydomain.OrganizationTargetRef, 0, len(targets))
	for index := range targets {
		results = append(results, *identityModelTargetRefToDomain(&targets[index]))
	}
	return results, total, nil
}

func (adapter *identityOrganizationStoreAdapter) ExistsByName(name string, excludeID ...int) (bool, error) {
	return adapter.repo.ExistsByName(name, excludeID...)
}

func (adapter *identityOrganizationStoreAdapter) Create(org *identitydomain.Organization) error {
	modelOrg := identityDomainOrganizationToModel(org)
	if err := adapter.repo.Create(modelOrg); err != nil {
		return err
	}
	*org = *identityModelOrganizationToDomain(modelOrg)
	return nil
}

func (adapter *identityOrganizationStoreAdapter) Update(org *identitydomain.Organization) error {
	return adapter.repo.Update(identityDomainOrganizationToModel(org))
}

func (adapter *identityOrganizationStoreAdapter) SoftDelete(id int) error {
	return adapter.repo.SoftDelete(id)
}

func (adapter *identityOrganizationStoreAdapter) BulkSoftDelete(ids []int) (int64, error) {
	return adapter.repo.BulkSoftDelete(ids)
}

func (adapter *identityOrganizationStoreAdapter) BulkAddTargets(organizationID int, targetIDs []int) error {
	err := adapter.repo.BulkAddTargets(organizationID, targetIDs)
	if errors.Is(err, identityrepo.ErrTargetNotFound) {
		return identitydomain.ErrTargetNotFound
	}
	return err
}

func (adapter *identityOrganizationStoreAdapter) UnlinkTargets(organizationID int, targetIDs []int) (int64, error) {
	return adapter.repo.UnlinkTargets(organizationID, targetIDs)
}
