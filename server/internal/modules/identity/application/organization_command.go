package application

import (
	"context"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

type OrganizationCommandService struct {
	store OrganizationCommandStore
}

func NewOrganizationCommandService(store OrganizationCommandStore) *OrganizationCommandService {
	return &OrganizationCommandService{store: store}
}

func (service *OrganizationCommandService) CreateOrganization(ctx context.Context, name, description string) (*identitydomain.Organization, error) {
	_ = ctx

	organization := identitydomain.NewOrganization(name, description)

	exists, err := service.store.ExistsByName(organization.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrOrganizationExists
	}

	if err := service.store.Create(organization); err != nil {
		return nil, err
	}
	return organization, nil
}

func (service *OrganizationCommandService) UpdateOrganization(ctx context.Context, id int, name, description string) (*identitydomain.Organization, error) {
	_ = ctx

	organization, err := service.store.GetActiveByID(id)
	if err != nil {
		return nil, err
	}

	originalName := organization.Name
	organization.UpdateProfile(name, description)

	if originalName != organization.Name {
		exists, err := service.store.ExistsByName(organization.Name, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrOrganizationExists
		}
	}

	if err := service.store.Update(organization); err != nil {
		return nil, err
	}
	return organization, nil
}

func (service *OrganizationCommandService) DeleteOrganization(ctx context.Context, id int) error {
	_ = ctx

	if _, err := service.store.GetActiveByID(id); err != nil {
		return err
	}
	return service.store.SoftDelete(id)
}

func (service *OrganizationCommandService) BulkDeleteOrganizations(ctx context.Context, ids []int) (int64, error) {
	_ = ctx
	return service.store.BulkSoftDelete(ids)
}

func (service *OrganizationCommandService) LinkTargets(ctx context.Context, organizationID int, targetIDs []int) error {
	_ = ctx

	if _, err := service.store.GetActiveByID(organizationID); err != nil {
		return err
	}
	return service.store.BulkAddTargets(organizationID, targetIDs)
}

func (service *OrganizationCommandService) UnlinkTargets(ctx context.Context, organizationID int, targetIDs []int) (int64, error) {
	_ = ctx

	if _, err := service.store.GetActiveByID(organizationID); err != nil {
		return 0, err
	}
	return service.store.UnlinkTargets(organizationID, targetIDs)
}
