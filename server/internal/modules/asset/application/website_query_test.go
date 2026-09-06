package application

import (
	"context"
	"errors"
	"testing"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type websiteQueryStoreStub struct {
	items          []assetdomain.Website
	total          int64
	count          int64
	findErr        error
	countErr       error
	streamErr      error
	scannedErr     error
	listPage       int
	listPageSize   int
	listFilter     string
	listOrderBy    string
	optionField    string
	screenshots    []assetdomain.Screenshot
	screenshotErr  error
	screenshotID   int
	screenshotURLs []string
}

func (stub *websiteQueryStoreStub) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Website, int64, error) {
	_ = targetID
	stub.listPage = page
	stub.listPageSize = pageSize
	stub.listFilter = filter
	stub.listOrderBy = orderBy
	if stub.findErr != nil {
		return nil, 0, stub.findErr
	}
	return append([]assetdomain.Website(nil), stub.items...), stub.total, nil
}

func (stub *websiteQueryStoreStub) GetByID(id int) (*assetdomain.Website, error) {
	if stub.findErr != nil {
		return nil, stub.findErr
	}
	for _, item := range stub.items {
		if item.ID == id {
			copyItem := item
			return &copyItem, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (stub *websiteQueryStoreStub) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	_ = targetID
	stub.optionField = field
	return []assetdomain.FilterOption{{Value: "nginx", Label: "nginx", Count: 2}}, nil
}

func (stub *websiteQueryStoreStub) ListSummariesByTargetAndURLs(targetID int, urls []string) ([]assetdomain.Screenshot, error) {
	stub.screenshotID = targetID
	stub.screenshotURLs = append([]string(nil), urls...)
	if stub.screenshotErr != nil {
		return nil, stub.screenshotErr
	}
	return append([]assetdomain.Screenshot(nil), stub.screenshots...), nil
}

func (stub *websiteQueryStoreStub) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Website) error) error {
	_ = ctx
	_ = targetID
	if stub.streamErr != nil {
		return stub.streamErr
	}
	if stub.scannedErr != nil {
		return stub.scannedErr
	}
	for _, item := range stub.items {
		if err := visit(item); err != nil {
			return err
		}
	}
	return nil
}

func (stub *websiteQueryStoreStub) CountByTargetID(targetID int) (int64, error) {
	_ = targetID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

type websiteTargetLookupQueryStub struct {
	target map[int]*assetdomain.TargetRef
	err    error
}

func (stub *websiteTargetLookupQueryStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	target, ok := stub.target[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyTarget := *target
	return &copyTarget, nil
}

func TestWebsiteQueryServiceListAndCount(t *testing.T) {
	store := &websiteQueryStoreStub{items: []assetdomain.Website{{ID: 1}}, total: 1, count: 3}
	lookup := &websiteTargetLookupQueryStub{target: map[int]*assetdomain.TargetRef{7: {ID: 7}}}
	service := NewWebsiteQueryService(store, store, lookup)

	result, err := service.ListByTarget(context.Background(), 7, WebsiteListQueryInput{PageSize: 20})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(result.Websites) != 1 || result.TotalSize != 1 {
		t.Fatalf("unexpected list result len=%d total=%d", len(result.Websites), result.TotalSize)
	}
	if store.listOrderBy != "createdAt desc" {
		t.Fatalf("expected default orderBy createdAt desc, got %q", store.listOrderBy)
	}

	count, err := service.CountByTarget(context.Background(), 7)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}
}

func TestWebsiteQueryServiceGetProjectsOnlyExactTargetAndURLScreenshot(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	status := int16(200)
	websiteURL := "https://api.acme.com/v1"
	store := &websiteQueryStoreStub{
		items: []assetdomain.Website{{ID: 11, TargetID: 7, URL: websiteURL}},
		screenshots: []assetdomain.Screenshot{
			{ID: 21, TargetID: 8, URL: websiteURL, StatusCode: &status, Image: []byte("wrong-target"), CreatedAt: now, UpdatedAt: now},
			{ID: 22, TargetID: 7, URL: "https://api.acme.com/other", StatusCode: &status, Image: []byte("wrong-url"), CreatedAt: now, UpdatedAt: now},
			{ID: 23, TargetID: 7, URL: websiteURL, StatusCode: &status, Image: []byte("image-must-not-project"), CreatedAt: now, UpdatedAt: now},
		},
	}
	service := NewWebsiteQueryService(store, store, &websiteTargetLookupQueryStub{})

	result, err := service.Get(context.Background(), 11)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if store.screenshotID != 7 || len(store.screenshotURLs) != 1 || store.screenshotURLs[0] != websiteURL {
		t.Fatalf("expected exact screenshot lookup, target=%d urls=%v", store.screenshotID, store.screenshotURLs)
	}
	if result.Screenshot == nil || result.Screenshot.ID != 23 || result.Screenshot.URL != websiteURL {
		t.Fatalf("expected exact screenshot projection, got %+v", result.Screenshot)
	}
	if result.Screenshot.StatusCode == nil || *result.Screenshot.StatusCode != status {
		t.Fatalf("expected screenshot status code, got %+v", result.Screenshot)
	}
}

func TestWebsiteQueryServiceGetHandlesAbsentAndMissingWebsite(t *testing.T) {
	store := &websiteQueryStoreStub{items: []assetdomain.Website{{ID: 11, TargetID: 7, URL: "https://api.acme.com"}}}
	service := NewWebsiteQueryService(store, store, &websiteTargetLookupQueryStub{})

	result, err := service.Get(context.Background(), 11)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result.Screenshot != nil {
		t.Fatalf("expected no screenshot projection, got %+v", result.Screenshot)
	}

	if _, err := service.Get(context.Background(), 404); !errors.Is(err, ErrWebsiteNotFound) {
		t.Fatalf("expected ErrWebsiteNotFound, got %v", err)
	}
}

func TestWebsiteQueryServiceTargetNotFound(t *testing.T) {
	store := &websiteQueryStoreStub{}
	lookup := &websiteTargetLookupQueryStub{err: gorm.ErrRecordNotFound}
	service := NewWebsiteQueryService(store, store, lookup)

	if err := service.ForEachByTarget(context.Background(), 1, func(assetdomain.Website) error { return nil }); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}

func TestWebsiteQueryServiceFilterOptions(t *testing.T) {
	store := &websiteQueryStoreStub{}
	lookup := &websiteTargetLookupQueryStub{target: map[int]*assetdomain.TargetRef{7: {ID: 7}}}
	service := NewWebsiteQueryService(store, store, lookup)

	options, err := service.ListFilterOptionsByTarget(context.Background(), 7, "webserver")
	if err != nil {
		t.Fatalf("filter options failed: %v", err)
	}
	if store.optionField != "webserver" || len(options) != 1 || options[0].Value != "nginx" {
		t.Fatalf("unexpected options field=%q options=%+v", store.optionField, options)
	}

	if _, err := service.ListFilterOptionsByTarget(context.Background(), 7, "url"); !errors.Is(err, ErrUnsupportedWebsiteFilter) {
		t.Fatalf("expected unsupported website filter, got %v", err)
	}

	lookup.err = gorm.ErrRecordNotFound
	if _, err := service.ListFilterOptionsByTarget(context.Background(), 7, "tech"); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected target not found, got %v", err)
	}
}

func TestWebsiteQueryServiceCanonicalControls(t *testing.T) {
	store := &websiteQueryStoreStub{items: []assetdomain.Website{{ID: 1}}, total: 21}
	lookup := &websiteTargetLookupQueryStub{target: map[int]*assetdomain.TargetRef{7: {ID: 7}}}
	service := NewWebsiteQueryService(store, store, lookup)

	result, err := service.ListByTarget(context.Background(), 7, WebsiteListQueryInput{
		PageSize: 10,
		Filter:   `url="admin" && (statusCode="200" || statusCode="301") && tech="nginx" && webserver="nginx" && contentType="text/html" && vhost="true"`,
		OrderBy:  "statusCode desc",
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if result.TotalSize != 21 || result.NextPageToken == "" {
		t.Fatalf("unexpected result total=%d token=%q", result.TotalSize, result.NextPageToken)
	}
	if store.listPage != 1 || store.listPageSize != 10 || store.listOrderBy != "statusCode desc" {
		t.Fatalf("unexpected list args page=%d size=%d orderBy=%q", store.listPage, store.listPageSize, store.listOrderBy)
	}

	if _, err := service.ListByTarget(context.Background(), 7, WebsiteListQueryInput{Filter: `status_code="200"`}); !errors.Is(err, ErrUnsupportedWebsiteFilter) {
		t.Fatalf("expected unsupported snake_case filter, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 7, WebsiteListQueryInput{OrderBy: "status_code desc"}); !errors.Is(err, ErrUnsupportedWebsiteOrderBy) {
		t.Fatalf("expected unsupported snake_case orderBy, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 7, WebsiteListQueryInput{Filter: `host="example.com"`}); !errors.Is(err, ErrUnsupportedWebsiteFilter) {
		t.Fatalf("expected unsupported host filter, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 7, WebsiteListQueryInput{OrderBy: "tech desc"}); !errors.Is(err, ErrUnsupportedWebsiteOrderBy) {
		t.Fatalf("expected unsupported tech orderBy, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 7, WebsiteListQueryInput{OrderBy: "url desc"}); !errors.Is(err, ErrUnsupportedWebsiteOrderBy) {
		t.Fatalf("expected unsupported url orderBy, got %v", err)
	}

	_, err = service.ListByTarget(context.Background(), 7, WebsiteListQueryInput{
		PageSize:  10,
		PageToken: result.NextPageToken,
		Filter:    `url="changed"`,
		OrderBy:   "statusCode desc",
	})
	if !errors.Is(err, ErrInvalidWebsitePageToken) {
		t.Fatalf("expected invalid token on query shape mismatch, got %v", err)
	}
}

func TestWebsiteFacadeForEachByTarget(t *testing.T) {
	store := &websiteFacadeStoreStub{
		websiteQueryStoreStub:   &websiteQueryStoreStub{items: []assetdomain.Website{{ID: 13}}},
		websiteCommandStoreStub: &websiteCommandStoreStub{},
	}
	lookup := &websiteTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1}}}
	facade := NewWebsiteFacade(NewWebsiteQueryService(store, store, lookup), NewWebsiteCommandService(store, lookup))

	var items []assetdomain.Website
	if err := facade.ForEachByTarget(1, func(item Website) error {
		items = append(items, item)
		return nil
	}); err != nil {
		t.Fatalf("for-each failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != 13 {
		t.Fatalf("unexpected items: %+v", items)
	}

	store.websiteQueryStoreStub.scannedErr = errors.New("scan failed")
	if err := facade.ForEachByTarget(1, func(Website) error { return nil }); err == nil || err.Error() != "scan failed" {
		t.Fatalf("expected scan error, got %v", err)
	}
}
