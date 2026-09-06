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
	screenshotSnapshotListDefaultPage     = 1
	screenshotSnapshotListDefaultPageSize = 20
	screenshotSnapshotListMaxPageSize     = 1000
	screenshotSnapshotListTokenVersion    = 1
)

var screenshotSnapshotQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"url":        {Column: "url", Exact: true},
	"statusCode": {Column: "status_code", IsNumeric: true},
})

var screenshotSnapshotOrderByFields = map[string]struct{}{
	"statusCode": {},
	"createdAt":  {},
}

type ScreenshotSnapshotListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type ScreenshotSnapshotListResult struct {
	Snapshots     []snapshotdomain.ScreenshotSnapshot
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type screenshotSnapshotListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	ScanID   int    `json:"sId"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
}

type ScreenshotSnapshotQueryService struct {
	store      ScreenshotSnapshotQueryStore
	scanLookup SnapshotScanRefLookup
}

func NewScreenshotSnapshotQueryService(store ScreenshotSnapshotQueryStore, scanLookup SnapshotScanRefLookup) *ScreenshotSnapshotQueryService {
	return &ScreenshotSnapshotQueryService{store: store, scanLookup: scanLookup}
}

func (service *ScreenshotSnapshotQueryService) ListByScan(ctx context.Context, scanID int, input ScreenshotSnapshotListQueryInput) (*ScreenshotSnapshotListResult, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}

	pageSize := normalizeScreenshotSnapshotListPageSize(input.PageSize)
	filter := scope.EmptyIfBlank(input.Filter)
	if err := validateScreenshotSnapshotListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeScreenshotSnapshotOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := screenshotSnapshotListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeScreenshotSnapshotListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.ScanID != scanID || payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidScreenshotSnapshotPageToken)
		}
		page = payload.Page
	}

	snapshots, total, err := service.store.ListByScanID(scanID, page, pageSize, filter, orderBy)
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeScreenshotSnapshotListPageToken(screenshotSnapshotListPageTokenPayload{
			Version:  screenshotSnapshotListTokenVersion,
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

	return &ScreenshotSnapshotListResult{
		Snapshots:     snapshots,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func (service *ScreenshotSnapshotQueryService) ListFilterOptionsByScan(ctx context.Context, scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}
	if field != "statusCode" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedScreenshotSnapshotFilter, field)
	}
	return service.store.ListFilterOptionsByScanID(scanID, field)
}

func normalizeScreenshotSnapshotListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return screenshotSnapshotListDefaultPageSize
	}
	if pageSize > screenshotSnapshotListMaxPageSize {
		return screenshotSnapshotListMaxPageSize
	}
	return pageSize
}

func validateScreenshotSnapshotListFilter(filter string) error {
	if err := scope.ValidateFilterDefault(filter, screenshotSnapshotQueryFilterMapping, "url"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedScreenshotSnapshotFilter, err)
	}
	return nil
}

func normalizeScreenshotSnapshotOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedScreenshotSnapshotOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := screenshotSnapshotOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedScreenshotSnapshotOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedScreenshotSnapshotOrderBy, orderBy)
	}
}

func encodeScreenshotSnapshotListPageToken(payload screenshotSnapshotListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = screenshotSnapshotListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeScreenshotSnapshotListPageToken(token string) (screenshotSnapshotListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return screenshotSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidScreenshotSnapshotPageToken)
	}
	var payload screenshotSnapshotListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return screenshotSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidScreenshotSnapshotPageToken)
	}
	if payload.Version != screenshotSnapshotListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 || payload.ScanID <= 0 {
		return screenshotSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidScreenshotSnapshotPageToken)
	}
	return payload, nil
}

func (service *ScreenshotSnapshotQueryService) GetByID(ctx context.Context, scanID int, id int) (*snapshotdomain.ScreenshotSnapshot, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}
	item, err := service.store.FindByIDAndScanID(id, scanID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrScreenshotSnapshotNotFound
		}
		return nil, err
	}
	return item, nil
}

type ScreenshotSnapshotCommandService struct {
	store       ScreenshotSnapshotCommandStore
	scanLookup  SnapshotCommandScanRefLookup
	assetSync   ScreenshotAssetSync
	coordinator MutableObservationMaterializationCoordinator
}

func NewScreenshotSnapshotCommandService(store ScreenshotSnapshotCommandStore, scanLookup SnapshotCommandScanRefLookup, assetSync ScreenshotAssetSync, coordinators ...MutableObservationMaterializationCoordinator) *ScreenshotSnapshotCommandService {
	service := &ScreenshotSnapshotCommandService{store: store, scanLookup: scanLookup, assetSync: assetSync}
	if len(coordinators) > 0 {
		service.coordinator = coordinators[0]
	}
	return service
}

func (service *ScreenshotSnapshotCommandService) SaveAndSync(ctx context.Context, scanID int, targetID int, items []ScreenshotSnapshotItem) (MaterializationSummary, error) {
	if service.coordinator == nil {
		return service.saveAndSync(ctx, scanID, targetID, items)
	}
	var summary MaterializationSummary
	err := service.coordinator.Materialize(ctx, scanID, targetID, func(txContext context.Context) error {
		var err error
		summary, err = service.saveAndSync(txContext, scanID, targetID, items)
		return err
	})
	return summary, err
}

// SaveResultBatchContext runs inside ResultIngestFacade's already-authorized
// transaction. Starting the normal coordinator here would create a separate
// transaction and let a late result outlive its Target/Task lease fence.
func (service *ScreenshotSnapshotCommandService) SaveResultBatchContext(ctx context.Context, scanID int, targetID int, items []ScreenshotSnapshotItem) (MaterializationSummary, error) {
	if ctx == nil {
		return MaterializationSummary{ReceivedItems: len(items)}, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return MaterializationSummary{ReceivedItems: len(items)}, err
	}
	return service.saveAndSync(ctx, scanID, targetID, items)
}

func (service *ScreenshotSnapshotCommandService) saveAndSync(ctx context.Context, scanID int, targetID int, items []ScreenshotSnapshotItem) (MaterializationSummary, error) {
	summary := MaterializationSummary{ReceivedItems: len(items)}
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

	accepted := make([]ScreenshotSnapshotItem, 0, len(items))
	for _, item := range items {
		if _, err := contractresults.ValidateObservedAssetURL(item.URL); err != nil {
			summary.InvalidItems++
			continue
		}
		if _, err := contractresults.DeriveObservedAssetURLHost(item.URL); err != nil {
			summary.InvalidItems++
			continue
		}
		if !isValidScreenshotImage(item.Image) {
			summary.InvalidItems++
			continue
		}
		if !snapshotdomain.IsURLMatchTarget(item.URL, *target) {
			summary.ScopeFilteredItems++
			continue
		}
		accepted = append(accepted, item)
	}
	accepted, duplicates := orderedLastByKey(accepted, func(item ScreenshotSnapshotItem) string { return item.URL })
	summary.DuplicateItems += duplicates
	snapshots := make([]snapshotdomain.ScreenshotSnapshot, 0, len(accepted))
	assetItems := make([]ScreenshotAssetItem, 0, len(accepted))
	for _, item := range accepted {
		snapshots = append(snapshots, snapshotdomain.ScreenshotSnapshot{ScanID: scanID, URL: item.URL, StatusCode: item.StatusCode, Image: item.Image})
		assetItems = append(assetItems, ScreenshotAssetItem(item))
	}

	if len(snapshots) == 0 {
		return summary, nil
	}
	summary.SnapshotCount, err = service.store.BatchUpsertContext(ctx, snapshots)
	if err != nil {
		return summary, err
	}
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	summary.AssetCount, err = service.assetSync.BatchUpsertContext(ctx, targetID, &ScreenshotAssetUpsertRequest{Screenshots: assetItems})
	if err != nil {
		return summary, err
	}
	return summary, nil
}
