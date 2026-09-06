package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type screenshotQueryStoreStub struct {
	items          []assetdomain.Screenshot
	total          int64
	itemByID       map[int]*assetdomain.Screenshot
	listErr        error
	findByIDErr    error
	listTargetID   int
	listPage       int
	listPageSize   int
	listFilter     string
	listOrderBy    string
	optionTargetID int
	optionField    string
}

func (stub *screenshotQueryStoreStub) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Screenshot, int64, error) {
	stub.listTargetID = targetID
	stub.listPage = page
	stub.listPageSize = pageSize
	stub.listFilter = filter
	stub.listOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]assetdomain.Screenshot(nil), stub.items...), stub.total, nil
}

func (stub *screenshotQueryStoreStub) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	stub.optionTargetID = targetID
	stub.optionField = field
	return []assetdomain.FilterOption{{Value: "200", Label: "200", Count: 2}}, nil
}

func (stub *screenshotQueryStoreStub) GetByID(id int) (*assetdomain.Screenshot, error) {
	if stub.findByIDErr != nil {
		return nil, stub.findByIDErr
	}
	item, ok := stub.itemByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyItem := *item
	return &copyItem, nil
}

type screenshotTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
}

func (stub *screenshotTargetLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	target, ok := stub.targets[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyTarget := *target
	return &copyTarget, nil
}

func TestScreenshotQueryServiceListAndGetByID(t *testing.T) {
	store := &screenshotQueryStoreStub{
		items:    []assetdomain.Screenshot{{ID: 1}, {ID: 2}},
		total:    2,
		itemByID: map[int]*assetdomain.Screenshot{2: {ID: 2}},
	}
	lookup := &screenshotTargetLookupStub{targets: map[int]*assetdomain.TargetRef{5: {ID: 5}}}
	service := NewScreenshotQueryService(store, lookup)

	result, err := service.ListByTarget(context.Background(), 5, ScreenshotListQueryInput{PageSize: 20})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(result.Screenshots) != 2 || result.TotalSize != 2 {
		t.Fatalf("unexpected list result len=%d total=%d", len(result.Screenshots), result.TotalSize)
	}
	if store.listTargetID != 5 || store.listPage != 1 || store.listPageSize != 20 || store.listOrderBy != "createdAt desc" {
		t.Fatalf("unexpected list args: %+v", store)
	}

	item, err := service.GetByID(context.Background(), 2)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if item.ID != 2 {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestScreenshotQueryServiceErrors(t *testing.T) {
	service := NewScreenshotQueryService(&screenshotQueryStoreStub{itemByID: map[int]*assetdomain.Screenshot{}}, &screenshotTargetLookupStub{err: gorm.ErrRecordNotFound})

	_, err := service.ListByTarget(context.Background(), 1, ScreenshotListQueryInput{PageSize: 20})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}

	_, err = service.GetByID(context.Background(), 99)
	if !errors.Is(err, ErrScreenshotNotFound) {
		t.Fatalf("expected ErrScreenshotNotFound, got %v", err)
	}
}

func TestScreenshotQueryServiceCanonicalControls(t *testing.T) {
	store := &screenshotQueryStoreStub{items: []assetdomain.Screenshot{{ID: 1}}, total: 21}
	lookup := &screenshotTargetLookupStub{targets: map[int]*assetdomain.TargetRef{7: {ID: 7}}}
	service := NewScreenshotQueryService(store, lookup)

	result, err := service.ListByTarget(context.Background(), 7, ScreenshotListQueryInput{
		PageSize: 10,
		Filter:   `url="admin" && (statusCode="200" || statusCode="301")`,
		OrderBy:  "statusCode desc",
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if result.TotalSize != 21 || result.NextPageToken == "" {
		t.Fatalf("unexpected result total=%d token=%q", result.TotalSize, result.NextPageToken)
	}
	if store.listPage != 1 || store.listPageSize != 10 || store.listFilter != `url="admin" && (statusCode="200" || statusCode="301")` || store.listOrderBy != "statusCode desc" {
		t.Fatalf("unexpected list args page=%d size=%d filter=%q orderBy=%q", store.listPage, store.listPageSize, store.listFilter, store.listOrderBy)
	}

	if _, err := service.ListByTarget(context.Background(), 7, ScreenshotListQueryInput{Filter: `status_code="200"`}); !errors.Is(err, ErrUnsupportedScreenshotFilter) {
		t.Fatalf("expected unsupported snake_case filter, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 7, ScreenshotListQueryInput{Filter: `contentType="text/html"`}); !errors.Is(err, ErrUnsupportedScreenshotFilter) {
		t.Fatalf("expected unsupported contentType filter, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 7, ScreenshotListQueryInput{OrderBy: "created_at desc"}); !errors.Is(err, ErrUnsupportedScreenshotOrderBy) {
		t.Fatalf("expected unsupported snake_case orderBy, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 7, ScreenshotListQueryInput{OrderBy: "url desc"}); !errors.Is(err, ErrUnsupportedScreenshotOrderBy) {
		t.Fatalf("expected unsupported url orderBy, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 7, ScreenshotListQueryInput{OrderBy: "updatedAt desc"}); !errors.Is(err, ErrUnsupportedScreenshotOrderBy) {
		t.Fatalf("expected unsupported updatedAt orderBy, got %v", err)
	}

	_, err = service.ListByTarget(context.Background(), 7, ScreenshotListQueryInput{
		PageSize:  10,
		PageToken: result.NextPageToken,
		Filter:    `url="changed"`,
		OrderBy:   "statusCode desc",
	})
	if !errors.Is(err, ErrInvalidScreenshotPageToken) {
		t.Fatalf("expected invalid token on query shape mismatch, got %v", err)
	}
}

func TestScreenshotQueryServiceFilterOptions(t *testing.T) {
	store := &screenshotQueryStoreStub{}
	lookup := &screenshotTargetLookupStub{targets: map[int]*assetdomain.TargetRef{7: {ID: 7}}}
	service := NewScreenshotQueryService(store, lookup)

	options, err := service.ListFilterOptionsByTarget(context.Background(), 7, "statusCode")
	if err != nil {
		t.Fatalf("filter options failed: %v", err)
	}
	if len(options) != 1 || options[0].Value != "200" || store.optionTargetID != 7 || store.optionField != "statusCode" {
		t.Fatalf("unexpected options=%+v target=%d field=%q", options, store.optionTargetID, store.optionField)
	}

	if _, err := service.ListFilterOptionsByTarget(context.Background(), 7, "url"); !errors.Is(err, ErrUnsupportedScreenshotFilter) {
		t.Fatalf("expected unsupported filter option field, got %v", err)
	}
}
