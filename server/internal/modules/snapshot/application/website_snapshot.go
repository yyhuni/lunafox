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
	websiteSnapshotListDefaultPage     = 1
	websiteSnapshotListDefaultPageSize = 20
	websiteSnapshotListMaxPageSize     = 1000
	websiteSnapshotListTokenVersion    = 1
)

var websiteSnapshotQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"url":         {Column: "url", Exact: true},
	"statusCode":  {Column: "status_code", IsNumeric: true},
	"tech":        {Column: "tech", IsArray: true},
	"webserver":   {Column: "webserver"},
	"contentType": {Column: "content_type"},
	"vhost":       {Column: "vhost", NeedsCast: true},
})

var websiteSnapshotOrderByFields = map[string]struct{}{
	"statusCode":    {},
	"contentLength": {},
	"createdAt":     {},
}

type WebsiteSnapshotListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type WebsiteSnapshotListResult struct {
	Snapshots     []snapshotdomain.WebsiteSnapshot
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type websiteSnapshotListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	ScanID   int    `json:"n"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
	Cursor   string `json:"c,omitempty"`
	ID       int    `json:"i,omitempty"`
}

type WebsiteSnapshotQueryService struct {
	store      WebsiteSnapshotQueryStore
	scanLookup SnapshotScanRefLookup
}

func NewWebsiteSnapshotQueryService(store WebsiteSnapshotQueryStore, scanLookup SnapshotScanRefLookup) *WebsiteSnapshotQueryService {
	return &WebsiteSnapshotQueryService{store: store, scanLookup: scanLookup}
}

func (service *WebsiteSnapshotQueryService) ListByScan(ctx context.Context, scanID int, input WebsiteSnapshotListQueryInput) (*WebsiteSnapshotListResult, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}

	pageSize := normalizeWebsiteSnapshotListPageSize(input.PageSize)
	filter := scope.EmptyIfBlank(input.Filter)
	if err := validateWebsiteSnapshotListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeWebsiteSnapshotOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := websiteSnapshotListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeWebsiteSnapshotListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.ScanID != scanID || payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidWebsiteSnapshotPageToken)
		}
		page = payload.Page
	}

	snapshots, total, err := service.store.ListByScanID(scanID, page, pageSize, filter, orderBy)
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeWebsiteSnapshotListPageToken(websiteSnapshotListPageTokenPayload{
			Version:  websiteSnapshotListTokenVersion,
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

	return &WebsiteSnapshotListResult{
		Snapshots:     snapshots,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func normalizeWebsiteSnapshotListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return websiteSnapshotListDefaultPageSize
	}
	if pageSize > websiteSnapshotListMaxPageSize {
		return websiteSnapshotListMaxPageSize
	}
	return pageSize
}

func validateWebsiteSnapshotListFilter(filter string) error {
	if err := scope.ValidateFilterDefault(filter, websiteSnapshotQueryFilterMapping, "url"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedWebsiteSnapshotFilter, err)
	}
	return nil
}

func normalizeWebsiteSnapshotOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedWebsiteSnapshotOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := websiteSnapshotOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedWebsiteSnapshotOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedWebsiteSnapshotOrderBy, orderBy)
	}
}

func encodeWebsiteSnapshotListPageToken(payload websiteSnapshotListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = websiteSnapshotListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeWebsiteSnapshotListPageToken(token string) (websiteSnapshotListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return websiteSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidWebsiteSnapshotPageToken)
	}
	var payload websiteSnapshotListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return websiteSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidWebsiteSnapshotPageToken)
	}
	if payload.Version != websiteSnapshotListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 || payload.ScanID <= 0 {
		return websiteSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidWebsiteSnapshotPageToken)
	}
	return payload, nil
}

func (service *WebsiteSnapshotQueryService) ForEachByScan(ctx context.Context, scanID int, visit func(snapshotdomain.WebsiteSnapshot) error) error {
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrSnapshotScanNotFound
		}
		return err
	}
	return service.store.ForEachByScanID(ctx, scanID, visit)
}

func (service *WebsiteSnapshotQueryService) CountByScan(ctx context.Context, scanID int) (int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrSnapshotScanNotFound
		}
		return 0, err
	}
	return service.store.CountByScanID(scanID)
}

type WebsiteSnapshotCommandService struct {
	store      WebsiteSnapshotCommandStore
	scanLookup SnapshotCommandScanRefLookup
	assetSync  WebsiteAssetSync
}

func NewWebsiteSnapshotCommandService(store WebsiteSnapshotCommandStore, scanLookup SnapshotCommandScanRefLookup, assetSync WebsiteAssetSync) *WebsiteSnapshotCommandService {
	return &WebsiteSnapshotCommandService{store: store, scanLookup: scanLookup, assetSync: assetSync}
}

func (service *WebsiteSnapshotCommandService) SaveAndSync(ctx context.Context, scanID int, targetID int, items []WebsiteSnapshotItem) (MaterializationSummary, error) {
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

	accepted := make([]WebsiteSnapshotItem, 0, len(items))
	for _, item := range items {
		if err := contractresults.Validate(contractresults.ResultKindAssetWebsite, contractresults.Website{
			URL: item.URL, Host: item.Host, Title: item.Title, StatusCode: item.StatusCode,
			ContentLength: item.ContentLength, Location: item.Location, Webserver: item.Webserver,
			ContentType: item.ContentType, Tech: item.Tech, ResponseBody: item.ResponseBody,
			Vhost: item.Vhost, ResponseHeaders: item.ResponseHeaders,
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
	accepted, duplicates := orderedLastByKey(accepted, func(item WebsiteSnapshotItem) string { return item.URL })
	summary.DuplicateItems += duplicates

	snapshots := make([]snapshotdomain.WebsiteSnapshot, 0, len(accepted))
	validItems := make([]WebsiteAssetUpsertItem, 0, len(accepted))
	for _, item := range accepted {
		host := item.Host
		snapshots = append(snapshots, snapshotdomain.WebsiteSnapshot{ScanID: scanID, URL: item.URL, Host: host, Title: item.Title, StatusCode: item.StatusCode, ContentLength: item.ContentLength, Location: item.Location, Webserver: item.Webserver, ContentType: item.ContentType, Tech: item.Tech, ResponseBody: item.ResponseBody, Vhost: item.Vhost, ResponseHeaders: item.ResponseHeaders})
		validItems = append(validItems, WebsiteAssetUpsertItem{URL: item.URL, Host: host, Title: item.Title, StatusCode: item.StatusCode, ContentLength: item.ContentLength, Location: item.Location, Webserver: item.Webserver, ContentType: item.ContentType, Tech: item.Tech, ResponseBody: item.ResponseBody, Vhost: item.Vhost, ResponseHeaders: item.ResponseHeaders})
	}

	if len(snapshots) == 0 {
		return summary, nil
	}

	if err := ctx.Err(); err != nil {
		return summary, err
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
