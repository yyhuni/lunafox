package application

import (
	"context"
	"errors"

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
func NewOrganizationFacade(queryService *OrganizationQueryService, cmdService *OrganizationCommandService) *OrganizationFacade {
	return &OrganizationFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// CreateOrganization creates a new organization.
func (service *OrganizationFacade) CreateOrganization(req *dto.CreateOrganizationRequest) (*Organization, error) {
	return service.CreateOrganizationContext(context.Background(), req)
}

// CreateOrganizationContext preserves the caller-owned cancellation and
// deadline through normalization, duplicate checks, and the database commit.
func (service *OrganizationFacade) CreateOrganizationContext(ctx context.Context, req *dto.CreateOrganizationRequest) (*Organization, error) {
	org, err := service.cmdService.CreateOrganization(ctx, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, ErrOrganizationExists) {
			return nil, ErrOrganizationExists
		}
		return nil, err
	}
	return org, nil
}

// ListOrganizations returns paginated organizations with target count.
func (service *OrganizationFacade) ListOrganizations(query *dto.OrganizationListQuery) (*OrganizationListResult, error) {
	return service.ListOrganizationsContext(context.Background(), query)
}

// ListOrganizationsContext preserves the caller-owned cancellation and
// deadline for MCP and other request-scoped readers.
func (service *OrganizationFacade) ListOrganizationsContext(ctx context.Context, query *dto.OrganizationListQuery) (*OrganizationListResult, error) {
	if service == nil || service.queryService == nil {
		return nil, ErrInvalidOrganizationQuery
	}
	if query == nil {
		return nil, ErrUnsupportedOrganizationFilter
	}
	result, err := service.queryService.ListOrganizations(ctx, OrganizationListQueryInput{
		PageSize:  query.GetPageSize(),
		PageToken: query.PageToken,
		Filter:    query.Filter,
		OrderBy:   query.OrderBy,
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrInvalidOrganizationQuery
	}
	return result, nil
}

// GetOrganizationByID returns an organization by ID with target count.
func (service *OrganizationFacade) GetOrganizationByID(id int) (*OrganizationWithCount, error) {
	return service.GetOrganizationByIDContext(context.Background(), id)
}

// GetOrganizationByIDContext preserves the caller-owned cancellation and
// deadline for organization detail reads.
func (service *OrganizationFacade) GetOrganizationByIDContext(ctx context.Context, id int) (*OrganizationWithCount, error) {
	if service == nil || service.queryService == nil {
		return nil, ErrInvalidOrganizationQuery
	}
	org, err := service.queryService.GetOrganizationByID(ctx, id)
	if err != nil {
		return nil, mapOrganizationBoundaryError(err)
	}
	if org == nil {
		return nil, ErrOrganizationNotFound
	}

	return &OrganizationWithCount{
		Organization: org.Organization,
		TargetCount:  org.TargetCount,
	}, nil
}

// UpdateOrganization updates an organization.
func (service *OrganizationFacade) UpdateOrganization(id int, req *dto.UpdateOrganizationRequest) (*Organization, error) {
	org, err := service.cmdService.UpdateOrganization(context.Background(), id, req.DisplayName, req.Description)
	if err != nil {
		if isIdentityRecordNotFound(err) {
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
		return mapOrganizationBoundaryError(err)
	}
	return nil
}

// BatchDeleteOrganizations soft deletes multiple organizations.
func (service *OrganizationFacade) BatchDeleteOrganizations(ids []int) (int64, error) {
	return service.cmdService.BatchDeleteOrganizations(context.Background(), ids)
}

// ListOrganizationTargets returns paginated targets for an organization.
func (service *OrganizationFacade) ListOrganizationTargets(organizationID int, query *dto.TargetListQuery) ([]OrganizationTargetRef, int64, error) {
	return service.ListOrganizationTargetsContext(context.Background(), organizationID, query)
}

// ListOrganizationTargetsContext preserves the caller-owned cancellation and
// deadline for organization target collection reads.
func (service *OrganizationFacade) ListOrganizationTargetsContext(ctx context.Context, organizationID int, query *dto.TargetListQuery) ([]OrganizationTargetRef, int64, error) {
	if service == nil || service.queryService == nil {
		return nil, 0, ErrInvalidOrganizationQuery
	}
	if query == nil {
		return nil, 0, ErrUnsupportedOrganizationFilter
	}
	targets, total, err := service.queryService.ListOrganizationTargets(
		ctx,
		organizationID,
		query.GetPage(),
		query.GetPageSize(),
		query.Type,
		query.Filter,
	)
	if err != nil {
		return nil, 0, mapOrganizationBoundaryError(err)
	}

	results := make([]OrganizationTargetRef, 0, len(targets))
	results = append(results, targets...)
	return results, total, nil
}

// LinkOrganizationTargets adds targets to an organization.
func (service *OrganizationFacade) LinkOrganizationTargets(organizationID int, targetIDs []int) error {
	err := service.cmdService.LinkTargets(context.Background(), organizationID, targetIDs)
	if err != nil {
		return mapOrganizationTargetLinkBoundaryError(err)
	}
	return nil
}

// UnlinkOrganizationTargets removes targets from an organization.
func (service *OrganizationFacade) UnlinkOrganizationTargets(organizationID int, targetIDs []int) (int64, error) {
	unlinkedCount, err := service.cmdService.UnlinkTargets(context.Background(), organizationID, targetIDs)
	if err != nil {
		return 0, mapOrganizationBoundaryError(err)
	}
	return unlinkedCount, nil
}
