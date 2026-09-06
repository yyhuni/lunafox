package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

const (
	hostPortListDefaultPage     = 1
	hostPortListDefaultPageSize = 20
	hostPortListMaxPageSize     = 1000
	hostPortListTokenVersion    = 1
)

var hostPortQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"ip":   {Column: "ip"},
	"host": {Column: "host"},
	"port": {Column: "port", IsNumeric: true},
})

var hostPortOrderByFields = map[string]struct{}{
	"ip":        {},
	"createdAt": {},
}

type HostPortListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type HostPortListResult struct {
	HostPorts     []HostPortResponse
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type hostPortListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	TargetID int    `json:"t"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
	Cursor   string `json:"c,omitempty"`
	ID       int    `json:"i,omitempty"`
}

type HostPortResponse struct {
	IP        string
	Hosts     []string
	Ports     []int
	CreatedAt time.Time
}

type HostPortQueryService struct {
	store        HostPortQueryStore
	targetLookup HostPortTargetLookup
}

func NewHostPortQueryService(store HostPortQueryStore, targetLookup HostPortTargetLookup) *HostPortQueryService {
	return &HostPortQueryService{store: store, targetLookup: targetLookup}
}

func (service *HostPortQueryService) ListByTarget(ctx context.Context, targetID int, input HostPortListQueryInput) (*HostPortListResult, error) {
	if _, err := getAssetTargetForQuery(ctx, service.targetLookup, targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}

	pageSize := normalizeHostPortListPageSize(input.PageSize)
	filter := strings.TrimSpace(input.Filter)
	if err := validateHostPortListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeHostPortOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := hostPortListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeHostPortListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.TargetID != targetID || payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidHostPortPageToken)
		}
		page = payload.Page
	}

	ipRows, total, err := listHostPortIPsForQuery(ctx, service.store, targetID, page, pageSize, filter, orderBy)
	if err != nil {
		return nil, err
	}

	results := make([]HostPortResponse, 0, len(ipRows))
	for _, row := range ipRows {
		hosts, ports, err := listHostPortDetailsForQuery(ctx, service.store, targetID, row.IP, filter)
		if err != nil {
			return nil, err
		}

		results = append(results, HostPortResponse{
			IP:        row.IP,
			Hosts:     hosts,
			Ports:     ports,
			CreatedAt: timeutil.ToUTC(row.CreatedAt),
		})
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeHostPortListPageToken(hostPortListPageTokenPayload{
			Version:  hostPortListTokenVersion,
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

	return &HostPortListResult{
		HostPorts:     results,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func normalizeHostPortListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return hostPortListDefaultPageSize
	}
	if pageSize > hostPortListMaxPageSize {
		return hostPortListMaxPageSize
	}
	return pageSize
}

func validateHostPortListFilter(filter string) error {
	if err := scope.ValidateFilterDefault(filter, hostPortQueryFilterMapping, "ip"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedHostPortFilter, err)
	}
	for _, group := range scope.ParseFilter(filter) {
		if group.Filter.Operator != "=" && group.Filter.Operator != "==" {
			return fmt.Errorf("%w: unsupported operator", ErrUnsupportedHostPortFilter)
		}
	}
	return nil
}

func normalizeHostPortOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedHostPortOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := hostPortOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedHostPortOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedHostPortOrderBy, orderBy)
	}
}

func encodeHostPortListPageToken(payload hostPortListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = hostPortListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeHostPortListPageToken(token string) (hostPortListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return hostPortListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidHostPortPageToken)
	}
	var payload hostPortListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return hostPortListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidHostPortPageToken)
	}
	if payload.Version != hostPortListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 || payload.TargetID <= 0 {
		return hostPortListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidHostPortPageToken)
	}
	return payload, nil
}

func (service *HostPortQueryService) ForEachByTarget(ctx context.Context, targetID int, visit func(assetdomain.HostPort) error) error {
	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrTargetNotFound
		}
		return err
	}

	return service.store.ForEachByTargetID(ctx, targetID, visit)
}

func (service *HostPortQueryService) ForEachByTargetAndIPs(ctx context.Context, targetID int, ips []string, visit func(assetdomain.HostPort) error) error {
	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrTargetNotFound
		}
		return err
	}

	return service.store.ForEachByTargetIDAndIPs(ctx, targetID, ips, visit)
}

func (service *HostPortQueryService) CountByTarget(ctx context.Context, targetID int) (int64, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	return service.store.CountByTargetID(targetID)
}
