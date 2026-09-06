package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

const (
	organizationListDefaultPage     = 1
	organizationListDefaultPageSize = 20
	organizationListMaxPageSize     = 1000
	organizationListTokenVersion    = 1
)

var organizationQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"displayName": {Column: "organization.name"},
})

var organizationOrderByFields = map[string]struct{}{
	"displayName": {},
	"createdAt":   {},
}

type OrganizationListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type OrganizationListResult struct {
	Organizations []identitydomain.OrganizationWithTargetCount
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type organizationListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
	Cursor   string `json:"c,omitempty"`
	ID       int    `json:"i,omitempty"`
}

type OrganizationQueryService struct {
	store OrganizationQueryStore
}

func NewOrganizationQueryService(store OrganizationQueryStore) *OrganizationQueryService {
	return &OrganizationQueryService{store: store}
}

func (service *OrganizationQueryService) ListOrganizations(ctx context.Context, input OrganizationListQueryInput) (*OrganizationListResult, error) {
	pageSize := normalizeOrganizationListPageSize(input.PageSize)
	filter := strings.TrimSpace(input.Filter)
	if err := validateOrganizationListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeOrganizationOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := organizationListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeOrganizationListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidOrganizationPageToken)
		}
		page = payload.Page
	}

	var organizations []identitydomain.OrganizationWithTargetCount
	var total int64
	if store, ok := service.store.(OrganizationQueryStoreContext); ok {
		organizations, total, err = store.ListContext(ctx, page, pageSize, filter, orderBy)
	} else {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		organizations, total, err = service.store.List(page, pageSize, filter, orderBy)
	}
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeOrganizationListPageToken(organizationListPageTokenPayload{
			Version:  organizationListTokenVersion,
			Page:     page + 1,
			PageSize: pageSize,
			Filter:   filter,
			OrderBy:  orderBy,
		})
		if err != nil {
			return nil, err
		}
	}

	return &OrganizationListResult{
		Organizations: organizations,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func normalizeOrganizationListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return organizationListDefaultPageSize
	}
	if pageSize > organizationListMaxPageSize {
		return organizationListMaxPageSize
	}
	return pageSize
}

func validateOrganizationListFilter(filter string) error {
	if err := scope.ValidateFilterDefault(filter, organizationQueryFilterMapping, "displayName"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedOrganizationFilter, err)
	}
	return nil
}

func normalizeOrganizationOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedOrganizationOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := organizationOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedOrganizationOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedOrganizationOrderBy, orderBy)
	}
}

func encodeOrganizationListPageToken(payload organizationListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = organizationListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeOrganizationListPageToken(token string) (organizationListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return organizationListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidOrganizationPageToken)
	}
	var payload organizationListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return organizationListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidOrganizationPageToken)
	}
	if payload.Version != organizationListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 {
		return organizationListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidOrganizationPageToken)
	}
	return payload, nil
}

func (service *OrganizationQueryService) GetOrganizationByID(ctx context.Context, id int) (*identitydomain.OrganizationWithTargetCount, error) {
	if store, ok := service.store.(OrganizationQueryStoreContext); ok {
		return store.FindByIDWithCountContext(ctx, id)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return service.store.FindByIDWithCount(id)
}

func (service *OrganizationQueryService) ListOrganizationTargets(
	ctx context.Context,
	organizationID int,
	page, pageSize int,
	targetType, filter string,
) ([]identitydomain.OrganizationTargetRef, int64, error) {
	var err error
	if store, ok := service.store.(OrganizationQueryStoreContext); ok {
		_, err = store.GetActiveByIDContext(ctx, organizationID)
	} else {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, 0, ctxErr
		}
		_, err = service.store.GetActiveByID(organizationID)
	}
	if err != nil {
		return nil, 0, err
	}
	if store, ok := service.store.(OrganizationQueryStoreContext); ok {
		return store.ListTargetsByOrganizationIDContext(ctx, organizationID, page, pageSize, targetType, filter)
	}
	return service.store.ListTargetsByOrganizationID(organizationID, page, pageSize, targetType, filter)
}
