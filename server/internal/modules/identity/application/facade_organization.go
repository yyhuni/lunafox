package application

import (
	"context"
	"errors"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"

	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
)

// OrganizationWithCount is a service projection for organization list/detail response.
type OrganizationWithCount struct {
	Organization
	TargetCount int64 `json:"targetCount"`
}

// OrganizationFacade handles organization business logic.
type OrganizationFacade struct {
	queryService *OrganizationQueryService
	cmdService   *OrganizationCommandService
}

// NewOrganizationFacade creates a new organization service.
func NewOrganizationFacade(queryStore OrganizationQueryStore, commandStore OrganizationCommandStore) *OrganizationFacade {
	return &OrganizationFacade{
		queryService: NewOrganizationQueryService(queryStore),
		cmdService:   NewOrganizationCommandService(commandStore),
	}
}

// CreateOrganization creates a new organization.
func (service *OrganizationFacade) CreateOrganization(req *dto.CreateOrganizationRequest) (*Organization, error) {
	org, err := service.cmdService.CreateOrganization(context.Background(), req.Name, req.Description)
	if err != nil {
		if errors.Is(err, ErrOrganizationExists) {
			return nil, ErrOrganizationExists
		}
		return nil, err
	}
	return org, nil
}

// ListOrganizations returns paginated organizations with target count.
func (service *OrganizationFacade) ListOrganizations(query *dto.OrganizationListQuery) ([]OrganizationWithCount, int64, error) {
	orgs, total, err := service.queryService.ListOrganizations(context.Background(), query.GetPage(), query.GetPageSize(), query.Filter)
	if err != nil {
		return nil, 0, err
	}

	results := make([]OrganizationWithCount, 0, len(orgs))
	for index := range orgs {
		results = append(results, OrganizationWithCount{
			Organization: orgs[index].Organization,
			TargetCount:  orgs[index].TargetCount,
		})
	}
	return results, total, nil
}

// GetOrganizationByID returns an organization by ID with target count.
func (service *OrganizationFacade) GetOrganizationByID(id int) (*OrganizationWithCount, error) {
	org, err := service.queryService.GetOrganizationByID(context.Background(), id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrOrganizationNotFound
		}
		return nil, err
	}

	return &OrganizationWithCount{
		Organization: org.Organization,
		TargetCount:  org.TargetCount,
	}, nil
}

// UpdateOrganization updates an organization.
func (service *OrganizationFacade) UpdateOrganization(id int, req *dto.UpdateOrganizationRequest) (*Organization, error) {
	org, err := service.cmdService.UpdateOrganization(context.Background(), id, req.Name, req.Description)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrOrganizationNotFound
		}
		if errors.Is(err, ErrOrganizationExists) {
			return nil, ErrOrganizationExists
		}
		return nil, err
	}
	return org, nil
}

// DeleteOrganization soft deletes an organization.
func (service *OrganizationFacade) DeleteOrganization(id int) error {
	err := service.cmdService.DeleteOrganization(context.Background(), id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrOrganizationNotFound
		}
		return err
	}
	return nil
}

// BulkDeleteOrganizations soft deletes multiple organizations.
func (service *OrganizationFacade) BulkDeleteOrganizations(ids []int) (int64, error) {
	return service.cmdService.BulkDeleteOrganizations(context.Background(), ids)
}

// ListOrganizationTargets returns paginated targets for an organization.
func (service *OrganizationFacade) ListOrganizationTargets(organizationID int, query *dto.TargetListQuery) ([]OrganizationTargetRef, int64, error) {
	targets, total, err := service.queryService.ListOrganizationTargets(
		context.Background(),
		organizationID,
		query.GetPage(),
		query.GetPageSize(),
		query.Type,
		query.Filter,
	)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrOrganizationNotFound
		}
		return nil, 0, err
	}

	results := make([]OrganizationTargetRef, 0, len(targets))
	for index := range targets {
		results = append(results, targets[index])
	}
	return results, total, nil
}

// LinkOrganizationTargets adds targets to an organization.
func (service *OrganizationFacade) LinkOrganizationTargets(organizationID int, targetIDs []int) error {
	err := service.cmdService.LinkTargets(context.Background(), organizationID, targetIDs)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrOrganizationNotFound
		}
		if errors.Is(err, ErrTargetNotFound) {
			return ErrTargetNotFound
		}
		return err
	}
	return nil
}

// UnlinkOrganizationTargets removes targets from an organization.
func (service *OrganizationFacade) UnlinkOrganizationTargets(organizationID int, targetIDs []int) (int64, error) {
	unlinkedCount, err := service.cmdService.UnlinkTargets(context.Background(), organizationID, targetIDs)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrOrganizationNotFound
		}
		return 0, err
	}
	return unlinkedCount, nil
}
