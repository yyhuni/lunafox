package identitywiring

import (
	"context"
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
	return identityModelOrganizationToIdentityDomainOrganization(org), nil
}

func (adapter *identityOrganizationStoreAdapter) GetActiveByIDContext(ctx context.Context, id int) (*identitydomain.Organization, error) {
	org, err := adapter.repo.GetActiveByIDContext(ctx, id)
	if err != nil {
		return nil, err
	}
	return identityModelOrganizationToIdentityDomainOrganization(org), nil
}

func (adapter *identityOrganizationStoreAdapter) FindByIDWithCount(id int) (*identitydomain.OrganizationWithTargetCount, error) {
	org, err := adapter.repo.FindByIDWithCount(id)
	if err != nil {
		return nil, err
	}
	return identityRepositoryOrganizationWithCountToIdentityDomainOrganizationWithTargetCount(org), nil
}

func (adapter *identityOrganizationStoreAdapter) FindByIDWithCountContext(ctx context.Context, id int) (*identitydomain.OrganizationWithTargetCount, error) {
	org, err := adapter.repo.FindByIDWithCountContext(ctx, id)
	if err != nil {
		return nil, err
	}
	return identityRepositoryOrganizationWithCountToIdentityDomainOrganizationWithTargetCount(org), nil
}

func (adapter *identityOrganizationStoreAdapter) List(page, pageSize int, filter, orderBy string) ([]identitydomain.OrganizationWithTargetCount, int64, error) {
	orgs, total, err := adapter.repo.List(page, pageSize, filter, orderBy)
	if err != nil {
		return nil, 0, err
	}
	return identityRepositoryOrganizationWithCountsToIdentityDomainOrganizationWithTargetCounts(orgs), total, nil
}

func (adapter *identityOrganizationStoreAdapter) ListContext(ctx context.Context, page, pageSize int, filter, orderBy string) ([]identitydomain.OrganizationWithTargetCount, int64, error) {
	orgs, total, err := adapter.repo.ListContext(ctx, page, pageSize, filter, orderBy)
	if err != nil {
		return nil, 0, err
	}
	return identityRepositoryOrganizationWithCountsToIdentityDomainOrganizationWithTargetCounts(orgs), total, nil
}

func (adapter *identityOrganizationStoreAdapter) ListTargetsByOrganizationID(organizationID int, page, pageSize int, targetType, filter string) ([]identitydomain.OrganizationTargetRef, int64, error) {
	targets, total, err := adapter.repo.ListTargetsByOrganizationID(organizationID, page, pageSize, targetType, filter)
	if err != nil {
		return nil, 0, err
	}
	return identityModelTargetRefsToIdentityDomainTargetRefs(targets), total, nil
}

func (adapter *identityOrganizationStoreAdapter) ListTargetsByOrganizationIDContext(ctx context.Context, organizationID int, page, pageSize int, targetType, filter string) ([]identitydomain.OrganizationTargetRef, int64, error) {
	targets, total, err := adapter.repo.ListTargetsByOrganizationIDContext(ctx, organizationID, page, pageSize, targetType, filter)
	if err != nil {
		return nil, 0, err
	}
	return identityModelTargetRefsToIdentityDomainTargetRefs(targets), total, nil
}

func (adapter *identityOrganizationStoreAdapter) ExistsByName(name string, excludeID ...int) (bool, error) {
	return adapter.repo.ExistsByName(name, excludeID...)
}

func (adapter *identityOrganizationStoreAdapter) ExistsByNameContext(ctx context.Context, name string, excludeID ...int) (bool, error) {
	return adapter.repo.ExistsByNameContext(ctx, name, excludeID...)
}

func (adapter *identityOrganizationStoreAdapter) Create(org *identitydomain.Organization) error {
	return adapter.CreateContext(context.Background(), org)
}

func (adapter *identityOrganizationStoreAdapter) CreateContext(ctx context.Context, org *identitydomain.Organization) error {
	modelOrg := identityDomainOrganizationToIdentityModelOrganization(org)
	if err := adapter.repo.CreateContext(ctx, modelOrg); err != nil {
		return err
	}
	*org = *identityModelOrganizationToIdentityDomainOrganization(modelOrg)
	return nil
}

func (adapter *identityOrganizationStoreAdapter) Update(org *identitydomain.Organization) error {
	return adapter.UpdateContext(context.Background(), org)
}

func (adapter *identityOrganizationStoreAdapter) UpdateContext(ctx context.Context, org *identitydomain.Organization) error {
	return adapter.repo.UpdateContext(ctx, identityDomainOrganizationToIdentityModelOrganization(org))
}

func (adapter *identityOrganizationStoreAdapter) SoftDelete(id int) error {
	return adapter.SoftDeleteContext(context.Background(), id)
}

func (adapter *identityOrganizationStoreAdapter) SoftDeleteContext(ctx context.Context, id int) error {
	return adapter.repo.SoftDeleteContext(ctx, id)
}

func (adapter *identityOrganizationStoreAdapter) BatchSoftDelete(ids []int) (int64, error) {
	return adapter.BatchSoftDeleteContext(context.Background(), ids)
}

func (adapter *identityOrganizationStoreAdapter) BatchSoftDeleteContext(ctx context.Context, ids []int) (int64, error) {
	return adapter.repo.BatchSoftDeleteContext(ctx, ids)
}

func (adapter *identityOrganizationStoreAdapter) BatchAddTargets(organizationID int, targetIDs []int) error {
	return adapter.BatchAddTargetsContext(context.Background(), organizationID, targetIDs)
}

func (adapter *identityOrganizationStoreAdapter) BatchAddTargetsContext(ctx context.Context, organizationID int, targetIDs []int) error {
	err := adapter.repo.BatchAddTargetsContext(ctx, organizationID, targetIDs)
	if errors.Is(err, identityrepo.ErrTargetNotFound) {
		return identitydomain.ErrTargetNotFound
	}
	return err
}

func (adapter *identityOrganizationStoreAdapter) UnlinkTargets(organizationID int, targetIDs []int) (int64, error) {
	return adapter.UnlinkTargetsContext(context.Background(), organizationID, targetIDs)
}

func (adapter *identityOrganizationStoreAdapter) UnlinkTargetsContext(ctx context.Context, organizationID int, targetIDs []int) (int64, error) {
	return adapter.repo.UnlinkTargetsContext(ctx, organizationID, targetIDs)
}
