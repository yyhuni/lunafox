package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

const (
	targetListDefaultPage     = 1
	targetListDefaultPageSize = 20
	targetListMaxPageSize     = 1000
	targetListTokenVersion    = 1
)

var targetQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"displayName": {Column: "name"},
	"type":        {Column: "type"},
})

var targetOrderByFields = map[string]struct{}{
	"displayName":   {},
	"createdAt":     {},
	"lastScannedAt": {},
}

var targetFilterTypes = map[string]struct{}{
	catalogdomain.TargetTypeDomain: {},
	catalogdomain.TargetTypeIP:     {},
	catalogdomain.TargetTypeCIDR:   {},
}

type TargetListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type TargetListResult struct {
	Targets       []catalogdomain.Target
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type targetListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
	Cursor   string `json:"c,omitempty"`
	ID       int    `json:"i,omitempty"`
}

type TargetSummary struct {
	Subdomains      int64
	Websites        int64
	Endpoints       int64
	IPs             int64
	Directories     int64
	Screenshots     int64
	Vulnerabilities *VulnerabilitySummary
}

type VulnerabilitySummary struct {
	Total    int64
	Critical int64
	High     int64
	Medium   int64
	Low      int64
}

type TargetQueryService struct {
	store TargetQueryStore
}

func NewTargetQueryService(store TargetQueryStore) *TargetQueryService {
	return &TargetQueryService{store: store}
}

func (service *TargetQueryService) ListTargets(ctx context.Context, input TargetListQueryInput) (*TargetListResult, error) {
	if service == nil || service.store == nil {
		return nil, fmt.Errorf("target query store is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	pageSize := normalizeTargetListPageSize(input.PageSize)
	filter := strings.TrimSpace(input.Filter)
	if err := validateTargetListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeTargetOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := targetListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeTargetListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidTargetPageToken)
		}
		page = payload.Page
	}

	var targets []catalogdomain.Target
	var total int64
	if store, ok := service.store.(TargetQueryStoreContext); ok {
		targets, total, err = store.ListContext(ctx, page, pageSize, filter, orderBy)
	} else {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		targets, total, err = service.store.List(page, pageSize, filter, orderBy)
	}
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeTargetListPageToken(targetListPageTokenPayload{
			Version:  targetListTokenVersion,
			Page:     page + 1,
			PageSize: pageSize,
			Filter:   filter,
			OrderBy:  orderBy,
		})
		if err != nil {
			return nil, err
		}
	}

	return &TargetListResult{
		Targets:       targets,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func normalizeTargetListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return targetListDefaultPageSize
	}
	if pageSize > targetListMaxPageSize {
		return targetListMaxPageSize
	}
	return pageSize
}

func validateTargetListFilter(filter string) error {
	if err := scope.ValidateFilterDefault(filter, targetQueryFilterMapping, "displayName"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedTargetFilter, err)
	}
	for _, group := range scope.ParseFilter(filter) {
		if strings.ToLower(group.Filter.Field) != "type" {
			continue
		}
		if group.Filter.Operator != "==" {
			return fmt.Errorf("%w: type requires ==", ErrUnsupportedTargetFilter)
		}
		if _, ok := targetFilterTypes[group.Filter.Value]; !ok {
			return fmt.Errorf("%w: unsupported type", ErrUnsupportedTargetFilter)
		}
	}
	return nil
}

func normalizeTargetOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedTargetOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := targetOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedTargetOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedTargetOrderBy, orderBy)
	}
}

func encodeTargetListPageToken(payload targetListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = targetListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeTargetListPageToken(token string) (targetListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return targetListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidTargetPageToken)
	}
	var payload targetListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return targetListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidTargetPageToken)
	}
	if payload.Version != targetListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 {
		return targetListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidTargetPageToken)
	}
	return payload, nil
}

func (service *TargetQueryService) GetTargetByID(ctx context.Context, id int) (*catalogdomain.Target, error) {
	if service == nil || service.store == nil {
		return nil, fmt.Errorf("target query store is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	if store, ok := service.store.(TargetQueryStoreContext); ok {
		return store.GetActiveByIDContext(ctx, id)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return service.store.GetActiveByID(id)
}

func (service *TargetQueryService) GetTargetDetailByID(ctx context.Context, id int) (*catalogdomain.Target, *TargetSummary, error) {
	if service == nil || service.store == nil {
		return nil, nil, fmt.Errorf("target query store is not configured")
	}
	if ctx == nil {
		return nil, nil, context.Canceled
	}
	var target *catalogdomain.Target
	var err error
	if store, ok := service.store.(TargetQueryStoreContext); ok {
		target, err = store.GetActiveByIDContext(ctx, id)
	} else {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		target, err = service.store.GetActiveByID(id)
	}
	if err != nil {
		return nil, nil, err
	}
	if target == nil {
		return nil, nil, ErrTargetNotFound
	}

	var assetCounts *catalogdomain.TargetAssetCounts
	if store, ok := service.store.(TargetQueryStoreContext); ok {
		assetCounts, err = store.GetAssetCountsSummaryContext(ctx, id)
	} else {
		assetCounts, err = service.store.GetAssetCountsSummary(id)
	}
	if err != nil {
		return nil, nil, err
	}
	if assetCounts == nil {
		return nil, nil, ErrTargetSummaryUnavailable
	}
	var vulnCounts *catalogdomain.VulnerabilityCounts
	if store, ok := service.store.(TargetQueryStoreContext); ok {
		vulnCounts, err = store.GetVulnerabilityCountsSummaryContext(ctx, id)
	} else {
		vulnCounts, err = service.store.GetVulnerabilityCountsSummary(id)
	}
	if err != nil {
		return nil, nil, err
	}
	if vulnCounts == nil {
		return nil, nil, ErrTargetSummaryUnavailable
	}

	summary := &TargetSummary{
		Subdomains:  assetCounts.Subdomains,
		Websites:    assetCounts.Websites,
		Endpoints:   assetCounts.Endpoints,
		IPs:         assetCounts.IPs,
		Directories: assetCounts.Directories,
		Screenshots: assetCounts.Screenshots,
		Vulnerabilities: &VulnerabilitySummary{
			Total:    vulnCounts.Total,
			Critical: vulnCounts.Critical,
			High:     vulnCounts.High,
			Medium:   vulnCounts.Medium,
			Low:      vulnCounts.Low,
		},
	}

	return target, summary, nil
}
