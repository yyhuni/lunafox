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
	websiteListDefaultPage     = 1
	websiteListDefaultPageSize = 20
	websiteListMaxPageSize     = 1000
	websiteListTokenVersion    = 1
)

var websiteQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"url":         {Column: "url", Exact: true},
	"statusCode":  {Column: "status_code", IsNumeric: true},
	"tech":        {Column: "tech", IsArray: true},
	"webserver":   {Column: "webserver"},
	"contentType": {Column: "content_type"},
	"vhost":       {Column: "vhost", NeedsCast: true},
})

var websiteOrderByFields = map[string]struct{}{
	"statusCode":    {},
	"contentLength": {},
	"createdAt":     {},
}

type WebsiteListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type WebsiteListResult struct {
	Websites      []WebsiteReadModel
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type websiteListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	TargetID int    `json:"t"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
	Cursor   string `json:"c,omitempty"`
	ID       int    `json:"i,omitempty"`
}

type WebsiteQueryService struct {
	store           WebsiteQueryStore
	screenshotStore WebsiteScreenshotQueryStore
	targetLookup    WebsiteTargetLookup
}

var ErrWebsiteScreenshotStoreUnavailable = errors.New("Website screenshot query store unavailable")

func NewWebsiteQueryService(store WebsiteQueryStore, screenshotStore WebsiteScreenshotQueryStore, targetLookup WebsiteTargetLookup) *WebsiteQueryService {
	return &WebsiteQueryService{store: store, screenshotStore: screenshotStore, targetLookup: targetLookup}
}

func (service *WebsiteQueryService) ListByTarget(ctx context.Context, targetID int, input WebsiteListQueryInput) (*WebsiteListResult, error) {
	if _, err := getAssetTargetForQuery(ctx, service.targetLookup, targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}

	pageSize := normalizeWebsiteListPageSize(input.PageSize)
	filter := scope.EmptyIfBlank(input.Filter)
	if err := validateWebsiteListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeWebsiteOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := websiteListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeWebsiteListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.TargetID != targetID || payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidWebsitePageToken)
		}
		page = payload.Page
	}

	websites, total, err := listWebsitesForQuery(ctx, service.store, targetID, page, pageSize, filter, orderBy)
	if err != nil {
		return nil, err
	}
	readModels, err := service.withScreenshotSummaries(ctx, targetID, websites)
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeWebsiteListPageToken(websiteListPageTokenPayload{
			Version:  websiteListTokenVersion,
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

	return &WebsiteListResult{
		Websites:      readModels,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

// Get returns one Website and its optional exact Target+URL Screenshot projection.
func (service *WebsiteQueryService) Get(ctx context.Context, id int) (*WebsiteReadModel, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	website, err := service.store.GetByID(id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrWebsiteNotFound
		}
		return nil, err
	}
	models, err := service.withScreenshotSummaries(ctx, website.TargetID, []assetdomain.Website{*website})
	if err != nil {
		return nil, err
	}
	if len(models) != 1 {
		return nil, errors.New("Website query returned an invalid read model")
	}
	return &models[0], nil
}

func (service *WebsiteQueryService) withScreenshotSummaries(ctx context.Context, targetID int, websites []assetdomain.Website) ([]WebsiteReadModel, error) {
	if service.screenshotStore == nil {
		return nil, ErrWebsiteScreenshotStoreUnavailable
	}
	urls := make([]string, 0, len(websites))
	for _, website := range websites {
		urls = append(urls, website.URL)
	}
	screenshots, err := listWebsiteScreenshotSummaries(ctx, service.screenshotStore, targetID, urls)
	if err != nil {
		return nil, err
	}
	byURL := make(map[string]assetdomain.Screenshot, len(screenshots))
	for _, screenshot := range screenshots {
		if screenshot.TargetID != targetID {
			continue
		}
		byURL[screenshot.URL] = screenshot
	}
	result := make([]WebsiteReadModel, 0, len(websites))
	for _, website := range websites {
		model := WebsiteReadModel{Website: website}
		if screenshot, ok := byURL[website.URL]; ok {
			model.Screenshot = &WebsiteScreenshotSummary{
				ID:         screenshot.ID,
				URL:        screenshot.URL,
				StatusCode: screenshot.StatusCode,
				CreatedAt:  screenshot.CreatedAt,
				UpdatedAt:  screenshot.UpdatedAt,
			}
		}
		result = append(result, model)
	}
	return result, nil
}

func normalizeWebsiteListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return websiteListDefaultPageSize
	}
	if pageSize > websiteListMaxPageSize {
		return websiteListMaxPageSize
	}
	return pageSize
}

func validateWebsiteListFilter(filter string) error {
	if err := scope.ValidateFilterDefault(filter, websiteQueryFilterMapping, "url"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedWebsiteFilter, err)
	}
	return nil
}

func normalizeWebsiteOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedWebsiteOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := websiteOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedWebsiteOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedWebsiteOrderBy, orderBy)
	}
}

func encodeWebsiteListPageToken(payload websiteListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = websiteListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeWebsiteListPageToken(token string) (websiteListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return websiteListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidWebsitePageToken)
	}
	var payload websiteListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return websiteListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidWebsitePageToken)
	}
	if payload.Version != websiteListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 || payload.TargetID <= 0 {
		return websiteListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidWebsitePageToken)
	}
	return payload, nil
}

func (service *WebsiteQueryService) ForEachByTarget(ctx context.Context, targetID int, visit func(assetdomain.Website) error) error {
	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrTargetNotFound
		}
		return err
	}

	return service.store.ForEachByTargetID(ctx, targetID, visit)
}

func (service *WebsiteQueryService) CountByTarget(ctx context.Context, targetID int) (int64, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	return service.store.CountByTargetID(targetID)
}
