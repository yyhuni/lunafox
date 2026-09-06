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
	directorySnapshotListDefaultPage     = 1
	directorySnapshotListDefaultPageSize = 20
	directorySnapshotListMaxPageSize     = 1000
	directorySnapshotListTokenVersion    = 1
)

var directorySnapshotQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"url":         {Column: "url", Exact: true},
	"status":      {Column: "status", IsNumeric: true},
	"contentType": {Column: "content_type"},
})

var directorySnapshotOrderByFields = map[string]struct{}{
	"status":        {},
	"contentLength": {},
	"createdAt":     {},
}

type DirectorySnapshotListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type DirectorySnapshotListResult struct {
	Snapshots     []snapshotdomain.DirectorySnapshot
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type directorySnapshotListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	ScanID   int    `json:"sId"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
}

type DirectorySnapshotQueryService struct {
	store      DirectorySnapshotQueryStore
	scanLookup SnapshotScanRefLookup
}

func NewDirectorySnapshotQueryService(store DirectorySnapshotQueryStore, scanLookup SnapshotScanRefLookup) *DirectorySnapshotQueryService {
	return &DirectorySnapshotQueryService{store: store, scanLookup: scanLookup}
}

func (service *DirectorySnapshotQueryService) ListByScan(ctx context.Context, scanID int, input DirectorySnapshotListQueryInput) (*DirectorySnapshotListResult, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}

	pageSize := normalizeDirectorySnapshotListPageSize(input.PageSize)
	filter := scope.EmptyIfBlank(input.Filter)
	if err := validateDirectorySnapshotListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeDirectorySnapshotOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := directorySnapshotListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeDirectorySnapshotListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.ScanID != scanID || payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidDirectorySnapshotPageToken)
		}
		page = payload.Page
	}

	snapshots, total, err := service.store.ListByScanID(scanID, page, pageSize, filter, orderBy)
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeDirectorySnapshotListPageToken(directorySnapshotListPageTokenPayload{
			Version:  directorySnapshotListTokenVersion,
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

	return &DirectorySnapshotListResult{
		Snapshots:     snapshots,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func normalizeDirectorySnapshotListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return directorySnapshotListDefaultPageSize
	}
	if pageSize > directorySnapshotListMaxPageSize {
		return directorySnapshotListMaxPageSize
	}
	return pageSize
}

func validateDirectorySnapshotListFilter(filter string) error {
	if err := scope.ValidateFilterDefault(filter, directorySnapshotQueryFilterMapping, "url"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedDirectorySnapshotFilter, err)
	}
	return nil
}

func normalizeDirectorySnapshotOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedDirectorySnapshotOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := directorySnapshotOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedDirectorySnapshotOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedDirectorySnapshotOrderBy, orderBy)
	}
}

func encodeDirectorySnapshotListPageToken(payload directorySnapshotListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = directorySnapshotListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeDirectorySnapshotListPageToken(token string) (directorySnapshotListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return directorySnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidDirectorySnapshotPageToken)
	}
	var payload directorySnapshotListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return directorySnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidDirectorySnapshotPageToken)
	}
	if payload.Version != directorySnapshotListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 || payload.ScanID <= 0 {
		return directorySnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidDirectorySnapshotPageToken)
	}
	return payload, nil
}

func (service *DirectorySnapshotQueryService) ForEachByScan(ctx context.Context, scanID int, visit func(snapshotdomain.DirectorySnapshot) error) error {
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrSnapshotScanNotFound
		}
		return err
	}
	return service.store.ForEachByScanID(ctx, scanID, visit)
}

func (service *DirectorySnapshotQueryService) CountByScan(ctx context.Context, scanID int) (int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrSnapshotScanNotFound
		}
		return 0, err
	}
	return service.store.CountByScanID(scanID)
}

type DirectorySnapshotCommandService struct {
	store       DirectorySnapshotCommandStore
	scanLookup  SnapshotCommandScanRefLookup
	assetSync   DirectoryAssetSync
	coordinator MutableObservationMaterializationCoordinator
}

func NewDirectorySnapshotCommandService(store DirectorySnapshotCommandStore, scanLookup SnapshotCommandScanRefLookup, assetSync DirectoryAssetSync, coordinators ...MutableObservationMaterializationCoordinator) *DirectorySnapshotCommandService {
	service := &DirectorySnapshotCommandService{store: store, scanLookup: scanLookup, assetSync: assetSync}
	if len(coordinators) > 0 {
		service.coordinator = coordinators[0]
	}
	return service
}

func (service *DirectorySnapshotCommandService) SaveAndSync(ctx context.Context, scanID int, targetID int, items []DirectorySnapshotItem) (MaterializationSummary, error) {
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

// SaveResultBatchContext applies per-item Directory admission inside the
// transaction owned by ResultIngestFacade.
func (service *DirectorySnapshotCommandService) SaveResultBatchContext(ctx context.Context, scanID int, targetID int, items []DirectorySnapshotItem) (MaterializationSummary, error) {
	return service.saveAndSync(ctx, scanID, targetID, items)
}

func (service *DirectorySnapshotCommandService) saveAndSync(ctx context.Context, scanID int, targetID int, items []DirectorySnapshotItem) (MaterializationSummary, error) {
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

	accepted := make([]DirectorySnapshotItem, 0, len(items))
	for _, item := range items {
		if _, err := contractresults.ValidateObservedAssetURL(item.URL); err != nil {
			summary.InvalidItems++
			continue
		}
		if _, err := contractresults.DeriveObservedAssetURLHost(item.URL); err != nil {
			summary.InvalidItems++
			continue
		}
		if !snapshotdomain.IsURLMatchTarget(item.URL, *target) {
			summary.ScopeFilteredItems++
			continue
		}
		accepted = append(accepted, item)
	}
	accepted, duplicates := orderedLastByKey(accepted, func(item DirectorySnapshotItem) string { return item.URL })
	summary.DuplicateItems += duplicates
	snapshots := make([]snapshotdomain.DirectorySnapshot, 0, len(accepted))
	validItems := make([]DirectoryAssetUpsertItem, 0, len(accepted))
	for _, item := range accepted {
		snapshots = append(snapshots, snapshotdomain.DirectorySnapshot{ScanID: scanID, URL: item.URL, Status: item.Status, ContentLength: item.ContentLength, ContentType: item.ContentType, Duration: item.Duration})
		validItems = append(validItems, DirectoryAssetUpsertItem(item))
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
