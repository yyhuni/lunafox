package application

import (
	"context"
	"errors"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

type OrganizationCommandService struct {
	store OrganizationCommandStore
}

func NewOrganizationCommandService(store OrganizationCommandStore) *OrganizationCommandService {
	return &OrganizationCommandService{store: store}
}

func (service *OrganizationCommandService) CreateOrganization(ctx context.Context, name, description string) (*identitydomain.Organization, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	organization := identitydomain.NewOrganization(name, description)

	exists, err := service.store.ExistsByNameContext(ctx, organization.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrOrganizationExists
	}

	if err := service.store.CreateContext(ctx, organization); err != nil {
		if errors.Is(err, identitydomain.ErrOrganizationExists) {
			return nil, ErrOrganizationExists
		}
		return nil, err
	}
	return organization, nil
}

func (service *OrganizationCommandService) UpdateOrganization(ctx context.Context, id int, name, description string) (*identitydomain.Organization, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	organization, err := service.store.GetActiveByIDContext(ctx, id)
	if err != nil {
		return nil, err
	}

	originalName := organization.Name
	organization.UpdateProfile(name, description)

	if originalName != organization.Name {
		exists, err := service.store.ExistsByNameContext(ctx, organization.Name, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrOrganizationExists
		}
	}

	if err := service.store.UpdateContext(ctx, organization); err != nil {
		if errors.Is(err, identitydomain.ErrOrganizationExists) {
			return nil, ErrOrganizationExists
		}
		return nil, err
	}
	return organization, nil
}

func (service *OrganizationCommandService) DeleteOrganization(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := service.store.GetActiveByIDContext(ctx, id); err != nil {
		return err
	}
	return service.store.SoftDeleteContext(ctx, id)
}

func (service *OrganizationCommandService) BatchDeleteOrganizations(ctx context.Context, ids []int) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return service.store.BatchSoftDeleteContext(ctx, ids)
}

func (service *OrganizationCommandService) LinkTargets(ctx context.Context, organizationID int, targetIDs []int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := service.store.GetActiveByIDContext(ctx, organizationID); err != nil {
		return err
	}
	return service.store.BatchAddTargetsContext(ctx, organizationID, targetIDs)
}

func (service *OrganizationCommandService) UnlinkTargets(ctx context.Context, organizationID int, targetIDs []int) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	if _, err := service.store.GetActiveByIDContext(ctx, organizationID); err != nil {
		return 0, err
	}
	return service.store.UnlinkTargetsContext(ctx, organizationID, targetIDs)
}
