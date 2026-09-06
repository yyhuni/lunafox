package application

import (
	"context"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type endpointSnapshotQueryStoreStub struct {
	items      []snapshotdomain.EndpointSnapshot
	total      int64
	count      int64
	listErr    error
	forEachErr error
	countErr   error
	visitErr   error
}

func (stub *endpointSnapshotQueryStoreStub) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.EndpointSnapshot, int64, error) {
	_ = scanID
	_ = page
	_ = pageSize
	_ = filter
	_ = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]snapshotdomain.EndpointSnapshot(nil), stub.items...), stub.total, nil
}

func (stub *endpointSnapshotQueryStoreStub) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	_ = scanID
	_ = field
	return []snapshotdomain.FilterOption{{Value: "nginx", Label: "nginx", Count: 2}}, nil
}

func (stub *endpointSnapshotQueryStoreStub) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.EndpointSnapshot) error) error {
	_ = ctx
	_ = scanID
	if stub.forEachErr != nil {
		return stub.forEachErr
	}
	for _, item := range stub.items {
		if stub.visitErr != nil {
			return stub.visitErr
		}
		if err := visit(item); err != nil {
			return err
		}
	}
	return nil
}

func (stub *endpointSnapshotQueryStoreStub) CountByScanID(scanID int) (int64, error) {
	_ = scanID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func TestEndpointSnapshotQueryServiceListAndCount(t *testing.T) {
	store := &endpointSnapshotQueryStoreStub{items: []snapshotdomain.EndpointSnapshot{{ID: 1}}, total: 1, count: 5}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 6}}
	service := NewEndpointSnapshotQueryService(store, lookup)

	result, err := service.ListByScan(context.Background(), 6, EndpointSnapshotListQueryInput{PageSize: 20})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(result.Snapshots) != 1 || result.TotalSize != 1 || result.PageSize != 20 {
		t.Fatalf("unexpected list result: %+v", result)
	}

	count, err := service.CountByScan(context.Background(), 6)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected count 5, got %d", count)
	}
}

func TestEndpointSnapshotQueryServiceScanNotFound(t *testing.T) {
	store := &endpointSnapshotQueryStoreStub{}
	lookup := &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}
	service := NewEndpointSnapshotQueryService(store, lookup)

	_, err := service.ListByScan(context.Background(), 1, EndpointSnapshotListQueryInput{PageSize: 20})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}
}

func TestEndpointSnapshotQueryServiceRejectsUnsupportedQueryFields(t *testing.T) {
	service := NewEndpointSnapshotQueryService(
		&endpointSnapshotQueryStoreStub{},
		&snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1}},
	)

	if _, err := service.ListByScan(context.Background(), 1, EndpointSnapshotListQueryInput{Filter: `host="api.example.com"`}); !errors.Is(err, ErrUnsupportedEndpointSnapshotFilter) {
		t.Fatalf("expected unsupported filter, got %v", err)
	}
	if _, err := service.ListByScan(context.Background(), 1, EndpointSnapshotListQueryInput{Filter: `status_code="200"`}); !errors.Is(err, ErrUnsupportedEndpointSnapshotFilter) {
		t.Fatalf("expected unsupported snake_case filter, got %v", err)
	}
	if _, err := service.ListByScan(context.Background(), 1, EndpointSnapshotListQueryInput{OrderBy: "url"}); !errors.Is(err, ErrUnsupportedEndpointSnapshotOrderBy) {
		t.Fatalf("expected unsupported orderBy, got %v", err)
	}
	if _, err := service.ListByScan(context.Background(), 1, EndpointSnapshotListQueryInput{OrderBy: "created_at desc"}); !errors.Is(err, ErrUnsupportedEndpointSnapshotOrderBy) {
		t.Fatalf("expected unsupported snake_case orderBy, got %v", err)
	}
}

func TestEndpointSnapshotQueryServiceBindsPageTokenToScanAndQueryShape(t *testing.T) {
	store := &endpointSnapshotQueryStoreStub{items: []snapshotdomain.EndpointSnapshot{{ID: 1}}, total: 2}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1}}
	service := NewEndpointSnapshotQueryService(
		store,
		lookup,
	)

	first, err := service.ListByScan(context.Background(), 1, EndpointSnapshotListQueryInput{PageSize: 1, Filter: `url="admin"`, OrderBy: "statusCode"})
	if err != nil {
		t.Fatalf("first page failed: %v", err)
	}
	if first.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	lookup.scan = &snapshotdomain.ScanRef{ID: 2}
	if _, err := service.ListByScan(context.Background(), 2, EndpointSnapshotListQueryInput{PageSize: 1, PageToken: first.NextPageToken, Filter: `url="admin"`, OrderBy: "statusCode"}); !errors.Is(err, ErrInvalidEndpointSnapshotPageToken) {
		t.Fatalf("expected scan-bound token rejection, got %v", err)
	}
	lookup.scan = &snapshotdomain.ScanRef{ID: 1}
	if _, err := service.ListByScan(context.Background(), 1, EndpointSnapshotListQueryInput{PageSize: 1, PageToken: first.NextPageToken, Filter: `url="api"`, OrderBy: "statusCode"}); !errors.Is(err, ErrInvalidEndpointSnapshotPageToken) {
		t.Fatalf("expected query-shape token rejection, got %v", err)
	}
}
