package application

import (
	"context"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type directorySnapshotQueryStoreStub struct {
	items        []snapshotdomain.DirectorySnapshot
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
	optionScanID int
	optionField  string
}

func (stub *directorySnapshotQueryStoreStub) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.DirectorySnapshot, int64, error) {
	_ = scanID
	stub.listPage = page
	stub.listPageSize = pageSize
	stub.listFilter = filter
	stub.listOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]snapshotdomain.DirectorySnapshot(nil), stub.items...), stub.total, nil
}

func (stub *directorySnapshotQueryStoreStub) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	stub.optionScanID = scanID
	stub.optionField = field
	return []snapshotdomain.FilterOption{{Value: "text/html", Label: "text/html", Count: 2}}, nil
}

func (stub *directorySnapshotQueryStoreStub) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.DirectorySnapshot) error) error {
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

func TestDirectorySnapshotQueryServiceFilterOptions(t *testing.T) {
	store := &directorySnapshotQueryStoreStub{}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 7}}
	service := NewDirectorySnapshotQueryService(store, lookup)

	options, err := service.ListFilterOptionsByScan(context.Background(), 7, "contentType")
	if err != nil {
		t.Fatalf("filter options failed: %v", err)
	}
	if len(options) != 1 || options[0].Value != "text/html" || store.optionScanID != 7 || store.optionField != "contentType" {
		t.Fatalf("unexpected options=%+v scan=%d field=%q", options, store.optionScanID, store.optionField)
	}

	if _, err := service.ListFilterOptionsByScan(context.Background(), 7, "url"); !errors.Is(err, ErrUnsupportedDirectorySnapshotFilter) {
		t.Fatalf("expected unsupported filter option field, got %v", err)
	}

	notFoundService := NewDirectorySnapshotQueryService(store, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound})
	if _, err := notFoundService.ListFilterOptionsByScan(context.Background(), 7, "contentType"); !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}
}

func (stub *directorySnapshotQueryStoreStub) CountByScanID(scanID int) (int64, error) {
	_ = scanID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func TestDirectorySnapshotQueryServiceListAndCount(t *testing.T) {
	store := &directorySnapshotQueryStoreStub{items: []snapshotdomain.DirectorySnapshot{{ID: 1}}, total: 1, count: 3}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 9}}
	service := NewDirectorySnapshotQueryService(store, lookup)

	result, err := service.ListByScan(context.Background(), 9, DirectorySnapshotListQueryInput{PageSize: 20})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(result.Snapshots) != 1 || result.TotalSize != 1 {
		t.Fatalf("unexpected list result len=%d total=%d", len(result.Snapshots), result.TotalSize)
	}
	if store.listOrderBy != "createdAt desc" {
		t.Fatalf("expected default orderBy createdAt desc, got %q", store.listOrderBy)
	}

	count, err := service.CountByScan(context.Background(), 9)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}
}

func TestDirectorySnapshotQueryServiceScanNotFound(t *testing.T) {
	store := &directorySnapshotQueryStoreStub{}
	lookup := &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}
	service := NewDirectorySnapshotQueryService(store, lookup)

	_, err := service.ListByScan(context.Background(), 1, DirectorySnapshotListQueryInput{PageSize: 20})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}
}

func TestDirectorySnapshotQueryServiceCanonicalControls(t *testing.T) {
	store := &directorySnapshotQueryStoreStub{items: []snapshotdomain.DirectorySnapshot{{ID: 1}}, total: 12}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 7}}
	service := NewDirectorySnapshotQueryService(store, lookup)

	result, err := service.ListByScan(context.Background(), 7, DirectorySnapshotListQueryInput{
		PageSize: 5,
		Filter:   `url="admin" && (status="200" || status="301") && contentType="text/html"`,
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

	if _, err := service.ListByScan(context.Background(), 7, DirectorySnapshotListQueryInput{Filter: `content_type="text/html"`}); !errors.Is(err, ErrUnsupportedDirectorySnapshotFilter) {
		t.Fatalf("expected unsupported snake_case filter, got %v", err)
	}
	if _, err := service.ListByScan(context.Background(), 7, DirectorySnapshotListQueryInput{Filter: `webserver="nginx"`}); !errors.Is(err, ErrUnsupportedDirectorySnapshotFilter) {
		t.Fatalf("expected unsupported webserver filter, got %v", err)
	}
	if _, err := service.ListByScan(context.Background(), 7, DirectorySnapshotListQueryInput{OrderBy: "created_at desc"}); !errors.Is(err, ErrUnsupportedDirectorySnapshotOrderBy) {
		t.Fatalf("expected unsupported snake_case orderBy, got %v", err)
	}
	if _, err := service.ListByScan(context.Background(), 7, DirectorySnapshotListQueryInput{OrderBy: "url desc"}); !errors.Is(err, ErrUnsupportedDirectorySnapshotOrderBy) {
		t.Fatalf("expected unsupported url orderBy, got %v", err)
	}

	_, err = service.ListByScan(context.Background(), 7, DirectorySnapshotListQueryInput{
		PageSize:  5,
		PageToken: result.NextPageToken,
		Filter:    `url="changed"`,
		OrderBy:   "contentLength desc",
	})
	if !errors.Is(err, ErrInvalidDirectorySnapshotPageToken) {
		t.Fatalf("expected invalid token on query shape mismatch, got %v", err)
	}
}
