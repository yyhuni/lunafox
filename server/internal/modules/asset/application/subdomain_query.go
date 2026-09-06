package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

const (
	subdomainListDefaultPage     = 1
	subdomainListDefaultPageSize = 20
	subdomainListMaxPageSize     = 1000
	subdomainListTokenVersion    = 1
)

var subdomainQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"dnsName": {Column: "dns_name"},
})

var subdomainOrderByFields = map[string]struct{}{
	"dnsName":   {},
	"createdAt": {},
}

type SubdomainListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type SubdomainListResult struct {
	Subdomains    []assetdomain.Subdomain
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type subdomainListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	TargetID int    `json:"t"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
	Cursor   string `json:"c,omitempty"`
	ID       int    `json:"i,omitempty"`
}

type SubdomainQueryService struct {
	store        SubdomainQueryStore
	targetLookup SubdomainTargetLookup
}

func NewSubdomainQueryService(store SubdomainQueryStore, targetLookup SubdomainTargetLookup) *SubdomainQueryService {
	return &SubdomainQueryService{store: store, targetLookup: targetLookup}
}

func (service *SubdomainQueryService) ListByTarget(ctx context.Context, targetID int, input SubdomainListQueryInput) (*SubdomainListResult, error) {
	if _, err := getAssetTargetForQuery(ctx, service.targetLookup, targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}

	pageSize := normalizeSubdomainListPageSize(input.PageSize)
	filter := strings.TrimSpace(input.Filter)
	if err := validateSubdomainListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeSubdomainOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := subdomainListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeSubdomainListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.TargetID != targetID || payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidSubdomainPageToken)
		}
		page = payload.Page
	}

	subdomains, total, err := listSubdomainsForQuery(ctx, service.store, targetID, page, pageSize, filter, orderBy)
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeSubdomainListPageToken(subdomainListPageTokenPayload{
			Version:  subdomainListTokenVersion,
			Page:     page + 1,
			PageSize: pageSize,
			TargetID: targetID,
			Filter:   filter,
			OrderBy:  orderBy,
		})
		if err != nil {
			return nil, err
		}
	}

	return &SubdomainListResult{
		Subdomains:    subdomains,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func normalizeSubdomainListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return subdomainListDefaultPageSize
	}
	if pageSize > subdomainListMaxPageSize {
		return subdomainListMaxPageSize
	}
	return pageSize
}

func validateSubdomainListFilter(filter string) error {
	if err := scope.ValidateFilterDefault(filter, subdomainQueryFilterMapping, "dnsName"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedSubdomainFilter, err)
	}
	return nil
}

func normalizeSubdomainOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedSubdomainOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := subdomainOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedSubdomainOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedSubdomainOrderBy, orderBy)
	}
}

func encodeSubdomainListPageToken(payload subdomainListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = subdomainListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeSubdomainListPageToken(token string) (subdomainListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return subdomainListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidSubdomainPageToken)
	}
	var payload subdomainListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return subdomainListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidSubdomainPageToken)
	}
	if payload.Version != subdomainListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 || payload.TargetID <= 0 {
		return subdomainListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidSubdomainPageToken)
	}
	return payload, nil
}

func (service *SubdomainQueryService) ForEachByTarget(ctx context.Context, targetID int, visit func(assetdomain.Subdomain) error) error {
	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrTargetNotFound
		}
		return err
	}

	return service.store.ForEachByTargetID(ctx, targetID, visit)
}

func (service *SubdomainQueryService) CountByTarget(ctx context.Context, targetID int) (int64, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	return service.store.CountByTargetID(targetID)
}
