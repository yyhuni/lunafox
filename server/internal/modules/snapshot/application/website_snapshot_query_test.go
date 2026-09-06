package application

import (
	"context"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type websiteSnapshotQueryStoreStub struct {
	items        []snapshotdomain.WebsiteSnapshot
	total        int64
	count        int64
	listErr      error
	forEachErr   error
	countErr     error
	visitErr     error
	listPage     int
	listPageSize int
	listFilter   string
	listOrderBy  string
	optionField  string
}

func (stub *websiteSnapshotQueryStoreStub) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.WebsiteSnapshot, int64, error) {
	_ = scanID
	stub.listPage = page
	stub.listPageSize = pageSize
	stub.listFilter = filter
	stub.listOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]snapshotdomain.WebsiteSnapshot(nil), stub.items...), stub.total, nil
}

func (stub *websiteSnapshotQueryStoreStub) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	_ = scanID
	stub.optionField = field
	return []snapshotdomain.FilterOption{{Value: "nginx", Label: "nginx", Count: 2}}, nil
}

func (stub *websiteSnapshotQueryStoreStub) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.WebsiteSnapshot) error) error {
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

func (stub *websiteSnapshotQueryStoreStub) CountByScanID(scanID int) (int64, error) {
	_ = scanID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

type snapshotScanLookupStub struct {
	scan      *snapshotdomain.ScanRef
	target    *snapshotdomain.ScanTargetRef
	findErr   error
	targetErr error
}

func (stub *snapshotScanLookupStub) GetScanRefByID(id int) (*snapshotdomain.ScanRef, error) {
	_ = id
	if stub.findErr != nil {
		return nil, stub.findErr
	}
	if stub.scan == nil {
		return nil, gorm.ErrRecordNotFound
	}
	copyScan := *stub.scan
	return &copyScan, nil
}

func (stub *snapshotScanLookupStub) GetScanRefByIDContext(ctx context.Context, id int) (*snapshotdomain.ScanRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.GetScanRefByID(id)
}

func (stub *snapshotScanLookupStub) GetTargetRefByScanID(scanID int) (*snapshotdomain.ScanTargetRef, error) {
	_ = scanID
	if stub.targetErr != nil {
		return nil, stub.targetErr
	}
	if stub.target == nil {
		return nil, gorm.ErrRecordNotFound
	}
	copyTarget := *stub.target
	return &copyTarget, nil
}

func (stub *snapshotScanLookupStub) GetTargetRefByScanIDContext(ctx context.Context, scanID int) (*snapshotdomain.ScanTargetRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.GetTargetRefByScanID(scanID)
}

func TestWebsiteSnapshotQueryServiceListAndCount(t *testing.T) {
	store := &websiteSnapshotQueryStoreStub{
		items: []snapshotdomain.WebsiteSnapshot{{ID: 1}, {ID: 2}},
		total: 2,
		count: 9,
	}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 7}}
	service := NewWebsiteSnapshotQueryService(store, lookup)

	result, err := service.ListByScan(context.Background(), 7, WebsiteSnapshotListQueryInput{PageSize: 20})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(result.Snapshots) != 2 || result.TotalSize != 2 {
		t.Fatalf("unexpected list result len=%d total=%d", len(result.Snapshots), result.TotalSize)
	}
	if store.listOrderBy != "createdAt desc" {
		t.Fatalf("expected default orderBy createdAt desc, got %q", store.listOrderBy)
	}

	count, err := service.CountByScan(context.Background(), 7)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 9 {
		t.Fatalf("expected count 9, got %d", count)
	}
}

func TestWebsiteSnapshotQueryServiceScanNotFound(t *testing.T) {
	store := &websiteSnapshotQueryStoreStub{}
	lookup := &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}
	service := NewWebsiteSnapshotQueryService(store, lookup)

	_, err := service.ListByScan(context.Background(), 1, WebsiteSnapshotListQueryInput{PageSize: 20})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}
}

func TestWebsiteSnapshotQueryServiceCanonicalControls(t *testing.T) {
	store := &websiteSnapshotQueryStoreStub{items: []snapshotdomain.WebsiteSnapshot{{ID: 1}}, total: 12}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 7}}
	service := NewWebsiteSnapshotQueryService(store, lookup)

	result, err := service.ListByScan(context.Background(), 7, WebsiteSnapshotListQueryInput{
		PageSize: 5,
		Filter:   `url="admin" && (statusCode="200" || statusCode="301") && tech="nginx" && webserver="nginx" && contentType="text/html" && vhost="true"`,
		OrderBy:  "contentLength desc",
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if result.TotalSize != 12 || result.NextPageToken == "" {
		t.Fatalf("unexpected result total=%d token=%q", result.TotalSize, result.NextPageToken)
	}
	if store.listPage != 1 || store.listPageSize != 5 || store.listOrderBy != "contentLength desc" {
		t.Fatalf("unexpected list args page=%d size=%d orderBy=%q", store.listPage, store.listPageSize, store.listOrderBy)
	}

	if _, err := service.ListByScan(context.Background(), 7, WebsiteSnapshotListQueryInput{Filter: `status_code="200"`}); !errors.Is(err, ErrUnsupportedWebsiteSnapshotFilter) {
		t.Fatalf("expected unsupported snake_case filter, got %v", err)
	}
	if _, err := service.ListByScan(context.Background(), 7, WebsiteSnapshotListQueryInput{OrderBy: "created_at desc"}); !errors.Is(err, ErrUnsupportedWebsiteSnapshotOrderBy) {
		t.Fatalf("expected unsupported snake_case orderBy, got %v", err)
	}
	if _, err := service.ListByScan(context.Background(), 7, WebsiteSnapshotListQueryInput{OrderBy: "webserver desc"}); !errors.Is(err, ErrUnsupportedWebsiteSnapshotOrderBy) {
		t.Fatalf("expected unsupported webserver orderBy, got %v", err)
	}
	if _, err := service.ListByScan(context.Background(), 7, WebsiteSnapshotListQueryInput{OrderBy: "url desc"}); !errors.Is(err, ErrUnsupportedWebsiteSnapshotOrderBy) {
		t.Fatalf("expected unsupported url orderBy, got %v", err)
	}

	_, err = service.ListByScan(context.Background(), 7, WebsiteSnapshotListQueryInput{
		PageSize:  5,
		PageToken: result.NextPageToken,
		Filter:    `url="changed"`,
		OrderBy:   "contentLength desc",
	})
	if !errors.Is(err, ErrInvalidWebsiteSnapshotPageToken) {
		t.Fatalf("expected invalid token on query shape mismatch, got %v", err)
	}
}
