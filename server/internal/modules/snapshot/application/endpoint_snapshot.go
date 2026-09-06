package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

const (
	endpointSnapshotListDefaultPage     = 1
	endpointSnapshotListDefaultPageSize = 20
	endpointSnapshotListMaxPageSize     = 1000
	endpointSnapshotListTokenVersion    = 1
)

var endpointSnapshotQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"url":         {Column: "url", Exact: true},
	"statusCode":  {Column: "status_code", IsNumeric: true},
	"tech":        {Column: "tech", IsArray: true},
	"webserver":   {Column: "webserver"},
	"contentType": {Column: "content_type"},
	"vhost":       {Column: "vhost", NeedsCast: true},
})

var endpointSnapshotOrderByFields = map[string]struct{}{
	"statusCode":    {},
	"contentLength": {},
	"createdAt":     {},
}

type EndpointSnapshotListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type EndpointSnapshotListResult struct {
	Snapshots     []snapshotdomain.EndpointSnapshot
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type endpointSnapshotListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	ScanID   int    `json:"n"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
	Cursor   string `json:"c,omitempty"`
	ID       int    `json:"i,omitempty"`
}

type EndpointSnapshotQueryService struct {
	store      EndpointSnapshotQueryStore
	scanLookup SnapshotScanRefLookup
}

func NewEndpointSnapshotQueryService(store EndpointSnapshotQueryStore, scanLookup SnapshotScanRefLookup) *EndpointSnapshotQueryService {
	return &EndpointSnapshotQueryService{store: store, scanLookup: scanLookup}
}

func (service *EndpointSnapshotQueryService) ListByScan(ctx context.Context, scanID int, input EndpointSnapshotListQueryInput) (*EndpointSnapshotListResult, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}

	pageSize := normalizeEndpointSnapshotListPageSize(input.PageSize)
	filter := scope.EmptyIfBlank(input.Filter)
	if err := validateEndpointSnapshotListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeEndpointSnapshotOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := endpointSnapshotListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeEndpointSnapshotListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.ScanID != scanID || payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidEndpointSnapshotPageToken)
		}
		page = payload.Page
	}

	snapshots, total, err := service.store.ListByScanID(scanID, page, pageSize, filter, orderBy)
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeEndpointSnapshotListPageToken(endpointSnapshotListPageTokenPayload{
			Version:  endpointSnapshotListTokenVersion,
			Page:     page + 1,
			PageSize: pageSize,
			ScanID:   scanID,
			Filter:   filter,
			OrderBy:  orderBy,
		})
		if err != nil {
			return nil, err
		}
	}

	return &EndpointSnapshotListResult{
		Snapshots:     snapshots,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func (service *EndpointSnapshotQueryService) ListFilterOptionsByScan(ctx context.Context, scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}
	if _, ok := endpointSnapshotQueryFilterMapping[field]; !ok || field == "url" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedEndpointSnapshotFilter, field)
	}
	return service.store.ListFilterOptionsByScanID(scanID, field)
}

func normalizeEndpointSnapshotListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return endpointSnapshotListDefaultPageSize
	}
	if pageSize > endpointSnapshotListMaxPageSize {
		return endpointSnapshotListMaxPageSize
	}
	return pageSize
}

func validateEndpointSnapshotListFilter(filter string) error {
	if err := scope.ValidateFilterDefault(filter, endpointSnapshotQueryFilterMapping, "url"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedEndpointSnapshotFilter, err)
	}
	return nil
}

func normalizeEndpointSnapshotOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedEndpointSnapshotOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := endpointSnapshotOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedEndpointSnapshotOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedEndpointSnapshotOrderBy, orderBy)
	}
}

func encodeEndpointSnapshotListPageToken(payload endpointSnapshotListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = endpointSnapshotListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeEndpointSnapshotListPageToken(token string) (endpointSnapshotListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return endpointSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidEndpointSnapshotPageToken)
	}
	var payload endpointSnapshotListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return endpointSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidEndpointSnapshotPageToken)
	}
	if payload.Version != endpointSnapshotListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 || payload.ScanID <= 0 {
		return endpointSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidEndpointSnapshotPageToken)
	}
	return payload, nil
}

func (service *EndpointSnapshotQueryService) ForEachByScan(ctx context.Context, scanID int, visit func(snapshotdomain.EndpointSnapshot) error) error {
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrSnapshotScanNotFound
		}
		return err
	}
	return service.store.ForEachByScanID(ctx, scanID, visit)
}

func (service *EndpointSnapshotQueryService) CountByScan(ctx context.Context, scanID int) (int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrSnapshotScanNotFound
		}
		return 0, err
	}
	return service.store.CountByScanID(scanID)
}

type EndpointSnapshotCommandService struct {
	store      EndpointSnapshotCommandStore
	scanLookup SnapshotCommandScanRefLookup
	assetSync  EndpointAssetSync
}

func NewEndpointSnapshotCommandService(store EndpointSnapshotCommandStore, scanLookup SnapshotCommandScanRefLookup, assetSync EndpointAssetSync) *EndpointSnapshotCommandService {
	return &EndpointSnapshotCommandService{store: store, scanLookup: scanLookup, assetSync: assetSync}
}

func (service *EndpointSnapshotCommandService) SaveAndSync(ctx context.Context, scanID int, targetID int, items []EndpointSnapshotItem) (MaterializationSummary, error) {
	// ResultIngestFacade owns the surrounding transaction, including the final
	// Target/Scan/Task lease fence and scan-summary refresh. This command stays
	// transaction-neutral so its repository calls join that caller-owned scope.
	return service.saveAndSync(ctx, scanID, targetID, items)
}

func (service *EndpointSnapshotCommandService) saveAndSync(ctx context.Context, scanID int, targetID int, items []EndpointSnapshotItem) (MaterializationSummary, error) {
	summary := MaterializationSummary{ReceivedItems: len(items)}
	if ctx == nil {
		return summary, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	if len(items) == 0 {
		return summary, nil
	}

	scan, err := service.scanLookup.GetScanRefByIDContext(ctx, scanID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return summary, ErrSnapshotScanNotFound
		}
		return summary, err
	}
	if scan.TargetID != targetID {
		return summary, ErrSnapshotTargetMismatch
	}

	target, err := service.scanLookup.GetTargetRefByScanIDContext(ctx, scanID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return summary, ErrSnapshotScanNotFound
		}
		return summary, err
	}

	accepted := make([]EndpointSnapshotItem, 0, len(items))
	for _, item := range items {
		if err := contractresults.Validate(contractresults.ResultKindAssetEndpoint, contractresults.Endpoint{
			URL: item.URL, Host: item.Host, Title: item.Title, StatusCode: item.StatusCode,
			ContentLength: item.ContentLength, Location: item.Location, Webserver: item.Webserver,
			ContentType: item.ContentType, Tech: item.Tech, ResponseBody: item.ResponseBody,
			ResponseBodyTruncated: item.ResponseBodyTruncated, Vhost: item.Vhost,
			ResponseHeaders: item.ResponseHeaders, ResponseHeadersTruncated: item.ResponseHeadersTruncated,
		}); err != nil {
			summary.InvalidItems++
			continue
		}
		if !snapshotdomain.IsURLMatchTarget(item.URL, *target) {
			summary.ScopeFilteredItems++
			continue
		}
		accepted = append(accepted, item)
	}
	accepted, duplicates := orderedLastByKey(accepted, func(item EndpointSnapshotItem) string { return item.URL })
	summary.DuplicateItems += duplicates
	snapshots := make([]snapshotdomain.EndpointSnapshot, 0, len(accepted))
	validItems := make([]EndpointAssetUpsertItem, 0, len(accepted))
	for _, item := range accepted {
		host := item.Host
		snapshots = append(snapshots, snapshotdomain.EndpointSnapshot{ScanID: scanID, URL: item.URL, Host: host, Title: item.Title, StatusCode: item.StatusCode, ContentLength: item.ContentLength, Location: item.Location, Webserver: item.Webserver, ContentType: item.ContentType, Tech: item.Tech, ResponseBody: item.ResponseBody, ResponseBodyTruncated: item.ResponseBodyTruncated, Vhost: item.Vhost, ResponseHeaders: item.ResponseHeaders, ResponseHeadersTruncated: item.ResponseHeadersTruncated})
		validItems = append(validItems, EndpointAssetUpsertItem{URL: item.URL, Host: host, Title: item.Title, StatusCode: item.StatusCode, ContentLength: item.ContentLength, Location: item.Location, Webserver: item.Webserver, ContentType: item.ContentType, Tech: item.Tech, ResponseBody: item.ResponseBody, ResponseBodyTruncated: item.ResponseBodyTruncated, Vhost: item.Vhost, ResponseHeaders: item.ResponseHeaders, ResponseHeadersTruncated: item.ResponseHeadersTruncated})
	}

	if len(snapshots) == 0 {
		return summary, nil
	}
	summary.SnapshotCount, err = service.store.BatchCreateContext(ctx, snapshots)
	if err != nil {
		return summary, err
	}
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	summary.AssetCount, err = service.assetSync.BatchUpsertContext(ctx, targetID, validItems)
	if err != nil {
		return summary, err
	}
	return summary, nil
}
