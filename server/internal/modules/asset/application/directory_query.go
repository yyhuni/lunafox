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
	directoryListDefaultPage     = 1
	directoryListDefaultPageSize = 20
	directoryListMaxPageSize     = 1000
	directoryListTokenVersion    = 1
)

var directoryQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"url":         {Column: "url", Exact: true},
	"status":      {Column: "status", IsNumeric: true},
	"contentType": {Column: "content_type"},
})

var directoryOrderByFields = map[string]struct{}{
	"status":        {},
	"contentLength": {},
	"createdAt":     {},
}

type DirectoryListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type DirectoryListResult struct {
	Directories   []assetdomain.Directory
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type directoryListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	TargetID int    `json:"t"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
}

type DirectoryQueryService struct {
	store        DirectoryQueryStore
	targetLookup DirectoryTargetLookup
}

func NewDirectoryQueryService(store DirectoryQueryStore, targetLookup DirectoryTargetLookup) *DirectoryQueryService {
	return &DirectoryQueryService{store: store, targetLookup: targetLookup}
}

func (service *DirectoryQueryService) ListByTarget(ctx context.Context, targetID int, input DirectoryListQueryInput) (*DirectoryListResult, error) {
	if _, err := getAssetTargetForQuery(ctx, service.targetLookup, targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}

	pageSize := normalizeDirectoryListPageSize(input.PageSize)
	filter := scope.EmptyIfBlank(input.Filter)
	if err := validateDirectoryListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeDirectoryOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := directoryListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeDirectoryListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.TargetID != targetID || payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidDirectoryPageToken)
		}
		page = payload.Page
	}

	directories, total, err := listDirectoriesForQuery(ctx, service.store, targetID, page, pageSize, filter, orderBy)
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeDirectoryListPageToken(directoryListPageTokenPayload{
			Version:  directoryListTokenVersion,
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

	return &DirectoryListResult{
		Directories:   directories,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func normalizeDirectoryListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return directoryListDefaultPageSize
	}
	if pageSize > directoryListMaxPageSize {
		return directoryListMaxPageSize
	}
	return pageSize
}

func validateDirectoryListFilter(filter string) error {
	remaining, _, err := webscope.ExtractFilterScope(filter)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedDirectoryFilter, err)
	}
	if err := scope.ValidateFilterDefault(remaining, directoryQueryFilterMapping, "url"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedDirectoryFilter, err)
	}
	return nil
}

func normalizeDirectoryOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedDirectoryOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := directoryOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedDirectoryOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedDirectoryOrderBy, orderBy)
	}
}

func encodeDirectoryListPageToken(payload directoryListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = directoryListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeDirectoryListPageToken(token string) (directoryListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return directoryListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidDirectoryPageToken)
	}
	var payload directoryListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return directoryListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidDirectoryPageToken)
	}
	if payload.Version != directoryListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 || payload.TargetID <= 0 {
		return directoryListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidDirectoryPageToken)
	}
	return payload, nil
}

func (service *DirectoryQueryService) ForEachByTarget(ctx context.Context, targetID int, visit func(assetdomain.Directory) error) error {
	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrTargetNotFound
		}
		return err
	}

	return service.store.ForEachByTargetID(ctx, targetID, visit)
}

func (service *DirectoryQueryService) CountByTarget(ctx context.Context, targetID int) (int64, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	return service.store.CountByTargetID(targetID)
}
