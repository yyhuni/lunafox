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
	subdomainSnapshotListDefaultPage     = 1
	subdomainSnapshotListDefaultPageSize = 20
	subdomainSnapshotListMaxPageSize     = 1000
	subdomainSnapshotListTokenVersion    = 1
)

var subdomainSnapshotQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"dnsName": {Column: "dns_name"},
})

var subdomainSnapshotOrderByFields = map[string]struct{}{
	"dnsName":   {},
	"createdAt": {},
}

type SubdomainSnapshotListResult struct {
	Subdomains    []snapshotdomain.SubdomainSnapshot
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type subdomainSnapshotListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	ScanID   int    `json:"n"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
	Cursor   string `json:"c,omitempty"`
	ID       int    `json:"i,omitempty"`
}

type SubdomainSnapshotQueryService struct {
	store      SubdomainSnapshotQueryStore
	scanLookup SnapshotScanRefLookup
}

func NewSubdomainSnapshotQueryService(store SubdomainSnapshotQueryStore, scanLookup SnapshotScanRefLookup) *SubdomainSnapshotQueryService {
	return &SubdomainSnapshotQueryService{store: store, scanLookup: scanLookup}
}

func (service *SubdomainSnapshotQueryService) ListByScan(ctx context.Context, scanID int, input SubdomainSnapshotListQueryInput) (*SubdomainSnapshotListResult, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}

	pageSize := normalizeSubdomainSnapshotListPageSize(input.PageSize)
	filter := strings.TrimSpace(input.Filter)
	if err := validateSubdomainSnapshotListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeSubdomainSnapshotOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := subdomainSnapshotListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeSubdomainSnapshotListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.ScanID != scanID || payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidSubdomainSnapshotPageToken)
		}
		page = payload.Page
	}

	subdomains, total, err := service.store.ListByScanID(scanID, page, pageSize, filter, orderBy)
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeSubdomainSnapshotListPageToken(subdomainSnapshotListPageTokenPayload{
			Version:  subdomainSnapshotListTokenVersion,
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

	return &SubdomainSnapshotListResult{
		Subdomains:    subdomains,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func normalizeSubdomainSnapshotListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return subdomainSnapshotListDefaultPageSize
	}
	if pageSize > subdomainSnapshotListMaxPageSize {
		return subdomainSnapshotListMaxPageSize
	}
	return pageSize
}

func validateSubdomainSnapshotListFilter(filter string) error {
	if err := scope.ValidateFilterDefault(filter, subdomainSnapshotQueryFilterMapping, "dnsName"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedSubdomainSnapshotFilter, err)
	}
	return nil
}

func normalizeSubdomainSnapshotOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedSubdomainSnapshotOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := subdomainSnapshotOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedSubdomainSnapshotOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedSubdomainSnapshotOrderBy, orderBy)
	}
}

func encodeSubdomainSnapshotListPageToken(payload subdomainSnapshotListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = subdomainSnapshotListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeSubdomainSnapshotListPageToken(token string) (subdomainSnapshotListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return subdomainSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidSubdomainSnapshotPageToken)
	}
	var payload subdomainSnapshotListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return subdomainSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidSubdomainSnapshotPageToken)
	}
	if payload.Version != subdomainSnapshotListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 || payload.ScanID <= 0 {
		return subdomainSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidSubdomainSnapshotPageToken)
	}
	return payload, nil
}

func (service *SubdomainSnapshotQueryService) ForEachByScan(ctx context.Context, scanID int, visit func(snapshotdomain.SubdomainSnapshot) error) error {
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrSnapshotScanNotFound
		}
		return err
	}
	return service.store.ForEachByScanID(ctx, scanID, visit)
}

func (service *SubdomainSnapshotQueryService) CountByScan(ctx context.Context, scanID int) (int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrSnapshotScanNotFound
		}
		return 0, err
	}
	return service.store.CountByScanID(scanID)
}

type SubdomainSnapshotCommandService struct {
	store      SubdomainSnapshotCommandStore
	scanLookup SnapshotCommandScanRefLookup
	assetSync  SubdomainAssetSync
}

func NewSubdomainSnapshotCommandService(store SubdomainSnapshotCommandStore, scanLookup SnapshotCommandScanRefLookup, assetSync SubdomainAssetSync) *SubdomainSnapshotCommandService {
	return &SubdomainSnapshotCommandService{store: store, scanLookup: scanLookup, assetSync: assetSync}
}

func (service *SubdomainSnapshotCommandService) SaveAndSync(ctx context.Context, scanID int, targetID int, items []SubdomainSnapshotItem) (MaterializationSummary, error) {
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
	if !snapshotdomain.IsDomainTargetType(target.Type) {
		return summary, ErrSubdomainSnapshotInvalidTargetType
	}
	canonicalTargetRoot, targetRootOK := contractresults.NormalizeSubdomainDNSName(target.Name)
	if !targetRootOK {
		return summary, ErrSubdomainSnapshotInvalidTargetType
	}

	snapshots := make([]snapshotdomain.SubdomainSnapshot, 0, len(items))
	validDNSNames := make([]string, 0, len(items))
	acceptedDNSNames := make([]string, 0, len(items))
	for _, item := range items {
		if err := contractresults.Validate(contractresults.ResultKindAssetSubdomain, contractresults.Subdomain{DNSName: item.DNSName}); err != nil {
			summary.InvalidItems++
			continue
		}
		// The canonical projection is used only for scope comparison. The
		// accepted value is already validated and is carried to both stores as-is.
		canonicalDNSName, _ := contractresults.NormalizeSubdomainDNSName(item.DNSName)
		if !snapshotdomain.IsSubdomainMatchTarget(canonicalDNSName, *target) || canonicalDNSName == canonicalTargetRoot {
			summary.ScopeFilteredItems++
			continue
		}
		acceptedDNSNames = append(acceptedDNSNames, item.DNSName)
	}
	validDNSNames, duplicates := orderedLastByKey(acceptedDNSNames, func(name string) string { return name })
	summary.DuplicateItems += duplicates
	for _, dnsName := range validDNSNames {
		snapshots = append(snapshots, snapshotdomain.SubdomainSnapshot{ScanID: scanID, DNSName: dnsName})
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
	if summary.SnapshotCount >= 0 && summary.SnapshotCount <= int64(len(snapshots)) {
		summary.DuplicateItems += len(snapshots) - int(summary.SnapshotCount)
	}
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	assetCountInt, err := service.assetSync.BatchCreateContext(ctx, targetID, validDNSNames)
	if err != nil {
		return summary, err
	}
	summary.AssetCount = int64(assetCountInt)
	return summary, nil
}
