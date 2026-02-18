package application

import (
	"context"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

type OrganizationQueryService struct {
	store OrganizationQueryStore
}

func NewOrganizationQueryService(store OrganizationQueryStore) *OrganizationQueryService {
	return &OrganizationQueryService{store: store}
}

func (service *OrganizationQueryService) ListOrganizations(ctx context.Context, page, pageSize int, filter string) ([]identitydomain.OrganizationWithTargetCount, int64, error) {
	_ = ctx
	return service.store.FindAll(page, pageSize, filter)
}

func (service *OrganizationQueryService) GetOrganizationByID(ctx context.Context, id int) (*identitydomain.OrganizationWithTargetCount, error) {
	_ = ctx
	return service.store.FindByIDWithCount(id)
}

func (service *OrganizationQueryService) ListOrganizationTargets(
	ctx context.Context,
	organizationID int,
	page, pageSize int,
	targetType, filter string,
) ([]identitydomain.OrganizationTargetRef, int64, error) {
	_ = ctx

	if _, err := service.store.GetActiveByID(organizationID); err != nil {
		return nil, 0, err
	}

	return service.store.FindTargets(organizationID, page, pageSize, targetType, filter)
}
