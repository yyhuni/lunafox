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
	"github.com/yyhuni/lunafox/server/internal/pkg/webscope"
)

const (
	endpointListDefaultPage     = 1
	endpointListDefaultPageSize = 20
	endpointListMaxPageSize     = 1000
	endpointListTokenVersion    = 1
)

var endpointQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"url":         {Column: "url", Exact: true},
	"statusCode":  {Column: "status_code", IsNumeric: true},
	"tech":        {Column: "tech", IsArray: true},
	"webserver":   {Column: "webserver"},
	"contentType": {Column: "content_type"},
	"vhost":       {Column: "vhost", NeedsCast: true},
})

var endpointOrderByFields = map[string]struct{}{
	"statusCode":    {},
	"contentLength": {},
	"createdAt":     {},
}

type EndpointListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type EndpointListResult struct {
	Endpoints     []assetdomain.Endpoint
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type endpointListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	TargetID int    `json:"t"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
	Cursor   string `json:"c,omitempty"`
	ID       int    `json:"i,omitempty"`
}

type EndpointQueryService struct {
	store        EndpointQueryStore
	targetLookup EndpointTargetLookup
}

func NewEndpointQueryService(store EndpointQueryStore, targetLookup EndpointTargetLookup) *EndpointQueryService {
	return &EndpointQueryService{store: store, targetLookup: targetLookup}
}

func (service *EndpointQueryService) ListByTarget(ctx context.Context, targetID int, input EndpointListQueryInput) (*EndpointListResult, error) {
	if _, err := getAssetTargetForQuery(ctx, service.targetLookup, targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}

	pageSize := normalizeEndpointListPageSize(input.PageSize)
	filter := scope.EmptyIfBlank(input.Filter)
	if err := validateEndpointListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeEndpointOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := endpointListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeEndpointListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.TargetID != targetID || payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidEndpointPageToken)
		}
		page = payload.Page
	}

	endpoints, total, err := listEndpointsForQuery(ctx, service.store, targetID, page, pageSize, filter, orderBy)
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeEndpointListPageToken(endpointListPageTokenPayload{
			Version:  endpointListTokenVersion,
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

	return &EndpointListResult{
		Endpoints:     endpoints,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func (service *EndpointQueryService) ListFilterOptionsByTarget(ctx context.Context, targetID int, field string) ([]assetdomain.FilterOption, error) {
	_ = ctx
	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}
	if _, ok := endpointQueryFilterMapping[field]; !ok || field == "url" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedEndpointFilter, field)
	}
	return service.store.ListFilterOptionsByTargetID(targetID, field)
}

func normalizeEndpointListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return endpointListDefaultPageSize
	}
	if pageSize > endpointListMaxPageSize {
		return endpointListMaxPageSize
	}
	return pageSize
}

func validateEndpointListFilter(filter string) error {
	remaining, _, err := webscope.ExtractFilterScope(filter)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedEndpointFilter, err)
	}
	if err := scope.ValidateFilterDefault(remaining, endpointQueryFilterMapping, "url"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedEndpointFilter, err)
	}
	return nil
}

func normalizeEndpointOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedEndpointOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := endpointOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedEndpointOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedEndpointOrderBy, orderBy)
	}
}

func encodeEndpointListPageToken(payload endpointListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = endpointListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeEndpointListPageToken(token string) (endpointListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return endpointListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidEndpointPageToken)
	}
	var payload endpointListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return endpointListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidEndpointPageToken)
	}
	if payload.Version != endpointListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 || payload.TargetID <= 0 {
		return endpointListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidEndpointPageToken)
	}
	return payload, nil
}

func (service *EndpointQueryService) GetByID(ctx context.Context, id int) (*assetdomain.Endpoint, error) {
	_ = ctx
	return service.store.GetByID(id)
}

func (service *EndpointQueryService) ForEachByTarget(ctx context.Context, targetID int, visit func(assetdomain.Endpoint) error) error {
	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrTargetNotFound
		}
		return err
	}

	return service.store.ForEachByTargetID(ctx, targetID, visit)
}

func (service *EndpointQueryService) CountByTarget(ctx context.Context, targetID int) (int64, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	return service.store.CountByTargetID(targetID)
}
