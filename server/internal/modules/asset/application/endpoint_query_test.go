package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type endpointQueryStoreStub struct {
	items        []assetdomain.Endpoint
	total        int64
	count        int64
	itemByID     map[int]*assetdomain.Endpoint
	listErr      error
	findErr      error
	streamErr    error
	countErr     error
	scannedErr   error
	listTargetID int
	listPage     int
	listPageSize int
	listFilter   string
	listOrderBy  string
}

func (stub *endpointQueryStoreStub) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Endpoint, int64, error) {
	stub.listTargetID = targetID
	stub.listPage = page
	stub.listPageSize = pageSize
	stub.listFilter = filter
	stub.listOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]assetdomain.Endpoint(nil), stub.items...), stub.total, nil
}

func (stub *endpointQueryStoreStub) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	_ = targetID
	_ = field
	return []assetdomain.FilterOption{{Value: "nginx", Label: "nginx", Count: 2}}, nil
}

func (stub *endpointQueryStoreStub) GetByID(id int) (*assetdomain.Endpoint, error) {
	if stub.findErr != nil {
		return nil, stub.findErr
	}
	item, ok := stub.itemByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyItem := *item
	return &copyItem, nil
}

func (stub *endpointQueryStoreStub) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Endpoint) error) error {
	_ = ctx
	_ = targetID
	if stub.streamErr != nil {
		return stub.streamErr
	}
	for _, item := range stub.items {
		if stub.scannedErr != nil {
			return stub.scannedErr
		}
		if err := visit(item); err != nil {
			return err
		}
	}
	return nil
}

func (stub *endpointQueryStoreStub) CountByTargetID(targetID int) (int64, error) {
	_ = targetID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

type endpointQueryTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
}

func (stub *endpointQueryTargetLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
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

func TestEndpointQueryServiceListGetAndCount(t *testing.T) {
	store := &endpointQueryStoreStub{
		items:    []assetdomain.Endpoint{{ID: 1}, {ID: 2}},
		total:    2,
		count:    5,
		itemByID: map[int]*assetdomain.Endpoint{2: {ID: 2}},
	}
	lookup := &endpointQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{7: {ID: 7}}}
	service := NewEndpointQueryService(store, lookup)

	result, err := service.ListByTarget(context.Background(), 7, EndpointListQueryInput{PageSize: 20})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(result.Endpoints) != 2 || result.TotalSize != 2 || result.PageSize != 20 {
		t.Fatalf("unexpected list result: %+v", result)
	}

	item, err := service.GetByID(context.Background(), 2)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if item.ID != 2 {
		t.Fatalf("unexpected item: %+v", item)
	}

	count, err := service.CountByTarget(context.Background(), 7)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected count 5, got %d", count)
	}
}

func TestEndpointQueryServiceTargetNotFound(t *testing.T) {
	store := &endpointQueryStoreStub{}
	lookup := &endpointQueryTargetLookupStub{err: gorm.ErrRecordNotFound}
	service := NewEndpointQueryService(store, lookup)

	_, err := service.ListByTarget(context.Background(), 1, EndpointListQueryInput{PageSize: 20})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}

func TestEndpointQueryServiceRejectsUnsupportedQueryFields(t *testing.T) {
	service := NewEndpointQueryService(
		&endpointQueryStoreStub{},
		&endpointQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1}}},
	)

	if _, err := service.ListByTarget(context.Background(), 1, EndpointListQueryInput{Filter: `host="api.example.com"`}); !errors.Is(err, ErrUnsupportedEndpointFilter) {
		t.Fatalf("expected unsupported filter, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 1, EndpointListQueryInput{Filter: `status_code="200"`}); !errors.Is(err, ErrUnsupportedEndpointFilter) {
		t.Fatalf("expected unsupported snake_case filter, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 1, EndpointListQueryInput{OrderBy: "url"}); !errors.Is(err, ErrUnsupportedEndpointOrderBy) {
		t.Fatalf("expected unsupported orderBy, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 1, EndpointListQueryInput{OrderBy: "created_at desc"}); !errors.Is(err, ErrUnsupportedEndpointOrderBy) {
		t.Fatalf("expected unsupported snake_case orderBy, got %v", err)
	}
}

func TestEndpointQueryServiceBindsPageTokenToTargetAndQueryShape(t *testing.T) {
	store := &endpointQueryStoreStub{items: []assetdomain.Endpoint{{ID: 1}}, total: 2}
	service := NewEndpointQueryService(
		store,
		&endpointQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1}, 2: {ID: 2}}},
	)

	first, err := service.ListByTarget(context.Background(), 1, EndpointListQueryInput{PageSize: 1, Filter: `url="admin"`, OrderBy: "statusCode"})
	if err != nil {
		t.Fatalf("first page failed: %v", err)
	}
	if first.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	if _, err := service.ListByTarget(context.Background(), 2, EndpointListQueryInput{PageSize: 1, PageToken: first.NextPageToken, Filter: `url="admin"`, OrderBy: "statusCode"}); !errors.Is(err, ErrInvalidEndpointPageToken) {
		t.Fatalf("expected target-bound token rejection, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 1, EndpointListQueryInput{PageSize: 1, PageToken: first.NextPageToken, Filter: `url="api"`, OrderBy: "statusCode"}); !errors.Is(err, ErrInvalidEndpointPageToken) {
		t.Fatalf("expected query-shape token rejection, got %v", err)
	}
}

func TestEndpointQueryServiceWebsiteScopeValidationAndPageBinding(t *testing.T) {
	store := &endpointQueryStoreStub{items: []assetdomain.Endpoint{{ID: 1}}, total: 2}
	service := NewEndpointQueryService(store, &endpointQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1}}})
	filter := `websiteUrl=="https://api.acme.com/a" && statusCode=="200"`

	first, err := service.ListByTarget(context.Background(), 1, EndpointListQueryInput{PageSize: 1, Filter: filter})
	if err != nil {
		t.Fatalf("ListByTarget returned error: %v", err)
	}
	if store.listFilter != filter || first.TotalSize != 2 || first.NextPageToken == "" {
		t.Fatalf("unexpected scoped endpoint result=%+v filter=%q", first, store.listFilter)
	}
	if _, err := service.ListByTarget(context.Background(), 1, EndpointListQueryInput{PageSize: 1, PageToken: first.NextPageToken, Filter: `websiteUrl=="https://api.acme.com/b" && statusCode=="200"`}); !errors.Is(err, ErrInvalidEndpointPageToken) {
		t.Fatalf("expected Website scope token binding, got %v", err)
	}

	for _, invalid := range []string{
		`websiteUrl="https://api.acme.com"`,
		`websiteUrl=="https://api.acme.com" || statusCode=="200"`,
		`websiteUrl=="https://api.acme.com" && websiteUrl=="https://api.acme.com/a"`,
		`websiteUrl=="not-a-url"`,
	} {
		if _, err := service.ListByTarget(context.Background(), 1, EndpointListQueryInput{Filter: invalid}); !errors.Is(err, ErrUnsupportedEndpointFilter) {
			t.Fatalf("filter %q: expected ErrUnsupportedEndpointFilter, got %v", invalid, err)
		}
	}
}
