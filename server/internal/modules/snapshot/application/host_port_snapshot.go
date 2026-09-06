package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

const (
	hostPortSnapshotListDefaultPage     = 1
	hostPortSnapshotListDefaultPageSize = 20
	hostPortSnapshotListMaxPageSize     = 1000
	hostPortSnapshotListTokenVersion    = 1
)

var hostPortSnapshotQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"ip":   {Column: "ip"},
	"host": {Column: "host"},
	"port": {Column: "port", IsNumeric: true},
})

var hostPortSnapshotOrderByFields = map[string]struct{}{
	"ip":        {},
	"createdAt": {},
}

type HostPortSnapshotAggregate struct {
	IP        string
	Hosts     []string
	Ports     []int
	CreatedAt time.Time
}

type HostPortSnapshotListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type HostPortSnapshotListResult struct {
	HostPorts     []HostPortSnapshotAggregate
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type hostPortSnapshotListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	ScanID   int    `json:"n"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
	Cursor   string `json:"c,omitempty"`
	ID       int    `json:"i,omitempty"`
}

type HostPortSnapshotQueryService struct {
	store      HostPortSnapshotQueryStore
	scanLookup SnapshotScanRefLookup
}

func NewHostPortSnapshotQueryService(store HostPortSnapshotQueryStore, scanLookup SnapshotScanRefLookup) *HostPortSnapshotQueryService {
	return &HostPortSnapshotQueryService{store: store, scanLookup: scanLookup}
}

func (service *HostPortSnapshotQueryService) ListByScan(ctx context.Context, scanID int, input HostPortSnapshotListQueryInput) (*HostPortSnapshotListResult, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}

	pageSize := normalizeHostPortSnapshotListPageSize(input.PageSize)
	filter := strings.TrimSpace(input.Filter)
	if err := validateHostPortSnapshotListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeHostPortSnapshotOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := hostPortSnapshotListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeHostPortSnapshotListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.ScanID != scanID || payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidHostPortSnapshotPageToken)
		}
		page = payload.Page
	}

	ipRows, total, err := service.store.GetIPAggregation(scanID, page, pageSize, filter, orderBy)
	if err != nil {
		return nil, err
	}

	results := make([]HostPortSnapshotAggregate, 0, len(ipRows))
	for _, row := range ipRows {
		hosts, ports, err := service.store.GetHostsAndPortsByIP(scanID, row.IP, filter)
		if err != nil {
			return nil, err
		}
		results = append(results, HostPortSnapshotAggregate{
			IP:        row.IP,
			Hosts:     hosts,
			Ports:     ports,
			CreatedAt: timeutil.ToUTC(row.CreatedAt),
		})
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeHostPortSnapshotListPageToken(hostPortSnapshotListPageTokenPayload{
			Version:  hostPortSnapshotListTokenVersion,
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

	return &HostPortSnapshotListResult{
		HostPorts:     results,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func normalizeHostPortSnapshotListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return hostPortSnapshotListDefaultPageSize
	}
	if pageSize > hostPortSnapshotListMaxPageSize {
		return hostPortSnapshotListMaxPageSize
	}
	return pageSize
}

func validateHostPortSnapshotListFilter(filter string) error {
	if err := scope.ValidateFilterDefault(filter, hostPortSnapshotQueryFilterMapping, "ip"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedHostPortSnapshotFilter, err)
	}
	for _, group := range scope.ParseFilter(filter) {
		if group.Filter.Operator != "=" && group.Filter.Operator != "==" {
			return fmt.Errorf("%w: unsupported operator", ErrUnsupportedHostPortSnapshotFilter)
		}
	}
	return nil
}

func normalizeHostPortSnapshotOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedHostPortSnapshotOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := hostPortSnapshotOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedHostPortSnapshotOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedHostPortSnapshotOrderBy, orderBy)
	}
}

func encodeHostPortSnapshotListPageToken(payload hostPortSnapshotListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = hostPortSnapshotListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeHostPortSnapshotListPageToken(token string) (hostPortSnapshotListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return hostPortSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidHostPortSnapshotPageToken)
	}
	var payload hostPortSnapshotListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return hostPortSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidHostPortSnapshotPageToken)
	}
	if payload.Version != hostPortSnapshotListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 || payload.ScanID <= 0 {
		return hostPortSnapshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidHostPortSnapshotPageToken)
	}
	return payload, nil
}

func (service *HostPortSnapshotQueryService) ForEachByScan(ctx context.Context, scanID int, visit func(snapshotdomain.HostPortSnapshot) error) error {
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrSnapshotScanNotFound
		}
		return err
	}
	return service.store.ForEachByScanID(ctx, scanID, visit)
}

func (service *HostPortSnapshotQueryService) CountByScan(ctx context.Context, scanID int) (int64, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrSnapshotScanNotFound
		}
		return 0, err
	}
	return service.store.CountByScanID(scanID)
}

type HostPortSnapshotCommandService struct {
	store      HostPortSnapshotCommandStore
	scanLookup SnapshotCommandScanRefLookup
	assetSync  HostPortAssetSync
}

func NewHostPortSnapshotCommandService(store HostPortSnapshotCommandStore, scanLookup SnapshotCommandScanRefLookup, assetSync HostPortAssetSync) *HostPortSnapshotCommandService {
	return &HostPortSnapshotCommandService{store: store, scanLookup: scanLookup, assetSync: assetSync}
}

func (service *HostPortSnapshotCommandService) SaveAndSync(ctx context.Context, scanID int, targetID int, items []HostPortSnapshotItem) (MaterializationSummary, error) {
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

	accepted := make([]HostPortSnapshotItem, 0, len(items))
	for _, item := range items {
		// IPv6 is outside the HostPort result contract and remains an explicit
		// unsupported item. Other malformed values are invalid; neither class is
		// repaired or allowed to reach persistence.
		if isIPv6HostPortAddress(item.IP) {
			summary.UnsupportedItems++
			continue
		}
		if validateErr := contractresults.Validate(contractresults.ResultKindAssetHostPort, contractresults.HostPort{
			Host: item.Host,
			IP:   item.IP,
			Port: item.Port,
		}); validateErr != nil {
			summary.InvalidItems++
			continue
		}
		if !snapshotdomain.IsHostPortMatchTarget(item.Host, item.IP, *target) {
			summary.ScopeFilteredItems++
			continue
		}
		accepted = append(accepted, item)
	}
	accepted, duplicates := orderedLastByKey(accepted, func(item HostPortSnapshotItem) string {
		return item.Host + "\x00" + item.IP + "\x00" + strconv.Itoa(item.Port)
	})
	summary.DuplicateItems += duplicates
	snapshots := make([]snapshotdomain.HostPortSnapshot, 0, len(accepted))
	validItems := make([]HostPortAssetItem, 0, len(accepted))
	for _, item := range accepted {
		snapshots = append(snapshots, snapshotdomain.HostPortSnapshot{ScanID: scanID, Host: item.Host, IP: item.IP, Port: item.Port})
		validItems = append(validItems, HostPortAssetItem(item))
	}

	if len(snapshots) == 0 {
		return summary, nil
	}
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	summary.SnapshotCount, err = service.store.BatchCreateContext(ctx, snapshots)
	if err != nil {
		return summary, fmt.Errorf("failed to batch create snapshots: %w", err)
	}
	if summary.SnapshotCount >= 0 && summary.SnapshotCount <= int64(len(snapshots)) {
		summary.DuplicateItems += len(snapshots) - int(summary.SnapshotCount)
	}
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	summary.AssetCount, err = service.assetSync.BatchUpsertContext(ctx, targetID, validItems)
	if err != nil {
		return summary, fmt.Errorf("failed to sync to asset table: %w", err)
	}
	return summary, nil
}

func isIPv6HostPortAddress(value string) bool {
	parsed := net.ParseIP(value)
	return parsed != nil && parsed.To4() == nil
}
