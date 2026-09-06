package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

const (
	screenshotListDefaultPage     = 1
	screenshotListDefaultPageSize = 20
	screenshotListMaxPageSize     = 1000
	screenshotListTokenVersion    = 1
)

var (
	ErrScreenshotNotFound = errors.New("screenshot not found")

	screenshotQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
		"url":        {Column: "url", Exact: true},
		"statusCode": {Column: "status_code", IsNumeric: true},
	})

	screenshotOrderByFields = map[string]struct{}{
		"statusCode": {},
		"createdAt":  {},
	}
)

type ScreenshotListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type ScreenshotListResult struct {
	Screenshots   []assetdomain.Screenshot
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type screenshotListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	TargetID int    `json:"t"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
}

type ScreenshotQueryService struct {
	store        ScreenshotQueryStore
	targetLookup ScreenshotTargetLookup
}

func NewScreenshotQueryService(store ScreenshotQueryStore, targetLookup ScreenshotTargetLookup) *ScreenshotQueryService {
	return &ScreenshotQueryService{store: store, targetLookup: targetLookup}
}

func (service *ScreenshotQueryService) ListByTarget(ctx context.Context, targetID int, input ScreenshotListQueryInput) (*ScreenshotListResult, error) {
	if service == nil || service.store == nil || service.targetLookup == nil {
		return nil, fmt.Errorf("screenshot query dependencies are not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	var targetErr error
	var target *assetdomain.TargetRef
	if lookup, ok := service.targetLookup.(interface {
		GetActiveByIDContext(context.Context, int) (*assetdomain.TargetRef, error)
	}); ok {
		target, targetErr = lookup.GetActiveByIDContext(ctx, targetID)
	} else {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		target, targetErr = service.targetLookup.GetActiveByID(targetID)
	}
	if targetErr != nil {
		if dberrors.IsRecordNotFound(targetErr) {
			return nil, ErrTargetNotFound
		}
		return nil, targetErr
	}
	if target == nil {
		return nil, ErrTargetNotFound
	}

	pageSize := normalizeScreenshotListPageSize(input.PageSize)
	filter := scope.EmptyIfBlank(input.Filter)
	if err := validateScreenshotListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeScreenshotOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := screenshotListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeScreenshotListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.TargetID != targetID || payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidScreenshotPageToken)
		}
		page = payload.Page
	}

	var screenshots []assetdomain.Screenshot
	var total int64
	if store, ok := service.store.(ScreenshotQueryStoreContext); ok {
		screenshots, total, err = store.ListByTargetIDContext(ctx, targetID, page, pageSize, filter, orderBy)
	} else {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		screenshots, total, err = service.store.ListByTargetID(targetID, page, pageSize, filter, orderBy)
	}
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeScreenshotListPageToken(screenshotListPageTokenPayload{
			Version:  screenshotListTokenVersion,
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

	return &ScreenshotListResult{
		Screenshots:   screenshots,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func (service *ScreenshotQueryService) ListFilterOptionsByTarget(ctx context.Context, targetID int, field string) ([]assetdomain.FilterOption, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}
	if field != "statusCode" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedScreenshotFilter, field)
	}
	return service.store.ListFilterOptionsByTargetID(targetID, field)
}

func normalizeScreenshotListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return screenshotListDefaultPageSize
	}
	if pageSize > screenshotListMaxPageSize {
		return screenshotListMaxPageSize
	}
	return pageSize
}

func validateScreenshotListFilter(filter string) error {
	if err := scope.ValidateFilterDefault(filter, screenshotQueryFilterMapping, "url"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedScreenshotFilter, err)
	}
	return nil
}

func normalizeScreenshotOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedScreenshotOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := screenshotOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedScreenshotOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedScreenshotOrderBy, orderBy)
	}
}

func encodeScreenshotListPageToken(payload screenshotListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = screenshotListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeScreenshotListPageToken(token string) (screenshotListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return screenshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidScreenshotPageToken)
	}
	var payload screenshotListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return screenshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidScreenshotPageToken)
	}
	if payload.Version != screenshotListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 || payload.TargetID <= 0 {
		return screenshotListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidScreenshotPageToken)
	}
	return payload, nil
}

func (service *ScreenshotQueryService) GetByID(ctx context.Context, id int) (*assetdomain.Screenshot, error) {
	if service == nil || service.store == nil {
		return nil, fmt.Errorf("screenshot query store is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	var item *assetdomain.Screenshot
	var err error
	if store, ok := service.store.(ScreenshotQueryStoreContext); ok {
		item, err = store.GetByIDContext(ctx, id)
	} else {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		item, err = service.store.GetByID(id)
	}
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrScreenshotNotFound
		}
		return nil, err
	}
	if item == nil {
		return nil, ErrScreenshotNotFound
	}
	return item, nil
}
