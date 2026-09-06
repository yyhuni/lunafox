package application

import (
	"context"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type screenshotSnapshotQueryStoreStub struct {
	items        []snapshotdomain.ScreenshotSnapshot
	total        int64
	itemByID     map[int]*snapshotdomain.ScreenshotSnapshot
	listErr      error
	findErr      error
	listScanID   int
	listPage     int
	listPageSize int
	listFilter   string
	listOrderBy  string
	optionScanID int
	optionField  string
}

func (stub *screenshotSnapshotQueryStoreStub) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.ScreenshotSnapshot, int64, error) {
	stub.listScanID = scanID
	stub.listPage = page
	stub.listPageSize = pageSize
	stub.listFilter = filter
	stub.listOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]snapshotdomain.ScreenshotSnapshot(nil), stub.items...), stub.total, nil
}

func (stub *screenshotSnapshotQueryStoreStub) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	stub.optionScanID = scanID
	stub.optionField = field
	return []snapshotdomain.FilterOption{{Value: "200", Label: "200", Count: 2}}, nil
}

func (stub *screenshotSnapshotQueryStoreStub) FindByIDAndScanID(id int, scanID int) (*snapshotdomain.ScreenshotSnapshot, error) {
	_ = scanID
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

func TestScreenshotSnapshotQueryServiceListAndGetByID(t *testing.T) {
	store := &screenshotSnapshotQueryStoreStub{
		items:    []snapshotdomain.ScreenshotSnapshot{{ID: 1}},
		total:    1,
		itemByID: map[int]*snapshotdomain.ScreenshotSnapshot{2: {ID: 2}},
	}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 6}}
	service := NewScreenshotSnapshotQueryService(store, lookup)

	result, err := service.ListByScan(context.Background(), 6, ScreenshotSnapshotListQueryInput{PageSize: 20})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(result.Snapshots) != 1 || result.TotalSize != 1 {
		t.Fatalf("unexpected list result len=%d total=%d", len(result.Snapshots), result.TotalSize)
	}
	if store.listScanID != 6 || store.listPage != 1 || store.listPageSize != 20 || store.listOrderBy != "createdAt desc" {
		t.Fatalf("unexpected list args: %+v", store)
	}

	item, err := service.GetByID(context.Background(), 6, 2)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if item.ID != 2 {
		t.Fatalf("unexpected item id=%d", item.ID)
	}
}

func TestScreenshotSnapshotQueryServiceErrors(t *testing.T) {
	service := NewScreenshotSnapshotQueryService(&screenshotSnapshotQueryStoreStub{itemByID: map[int]*snapshotdomain.ScreenshotSnapshot{}}, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound})

	_, err := service.ListByScan(context.Background(), 1, ScreenshotSnapshotListQueryInput{PageSize: 20})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}

	service = NewScreenshotSnapshotQueryService(&screenshotSnapshotQueryStoreStub{itemByID: map[int]*snapshotdomain.ScreenshotSnapshot{}}, &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 1}})
	_, err = service.GetByID(context.Background(), 1, 99)
	if !errors.Is(err, ErrScreenshotSnapshotNotFound) {
		t.Fatalf("expected ErrScreenshotSnapshotNotFound, got %v", err)
	}
}

func TestScreenshotSnapshotQueryServiceCanonicalControls(t *testing.T) {
	store := &screenshotSnapshotQueryStoreStub{items: []snapshotdomain.ScreenshotSnapshot{{ID: 1}}, total: 21}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 7}}
	service := NewScreenshotSnapshotQueryService(store, lookup)

	result, err := service.ListByScan(context.Background(), 7, ScreenshotSnapshotListQueryInput{
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

	if _, err := service.ListByScan(context.Background(), 7, ScreenshotSnapshotListQueryInput{Filter: `status_code="200"`}); !errors.Is(err, ErrUnsupportedScreenshotSnapshotFilter) {
		t.Fatalf("expected unsupported snake_case filter, got %v", err)
	}
	if _, err := service.ListByScan(context.Background(), 7, ScreenshotSnapshotListQueryInput{Filter: `contentType="text/html"`}); !errors.Is(err, ErrUnsupportedScreenshotSnapshotFilter) {
		t.Fatalf("expected unsupported contentType filter, got %v", err)
	}
	if _, err := service.ListByScan(context.Background(), 7, ScreenshotSnapshotListQueryInput{OrderBy: "created_at desc"}); !errors.Is(err, ErrUnsupportedScreenshotSnapshotOrderBy) {
		t.Fatalf("expected unsupported snake_case orderBy, got %v", err)
	}
	if _, err := service.ListByScan(context.Background(), 7, ScreenshotSnapshotListQueryInput{OrderBy: "url desc"}); !errors.Is(err, ErrUnsupportedScreenshotSnapshotOrderBy) {
		t.Fatalf("expected unsupported url orderBy, got %v", err)
	}

	_, err = service.ListByScan(context.Background(), 7, ScreenshotSnapshotListQueryInput{
		PageSize:  10,
		PageToken: result.NextPageToken,
		Filter:    `url="changed"`,
		OrderBy:   "statusCode desc",
	})
	if !errors.Is(err, ErrInvalidScreenshotSnapshotPageToken) {
		t.Fatalf("expected invalid token on query shape mismatch, got %v", err)
	}
}

func TestScreenshotSnapshotQueryServiceFilterOptions(t *testing.T) {
	store := &screenshotSnapshotQueryStoreStub{}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 7}}
	service := NewScreenshotSnapshotQueryService(store, lookup)

	options, err := service.ListFilterOptionsByScan(context.Background(), 7, "statusCode")
	if err != nil {
		t.Fatalf("filter options failed: %v", err)
	}
	if len(options) != 1 || options[0].Value != "200" || store.optionScanID != 7 || store.optionField != "statusCode" {
		t.Fatalf("unexpected options=%+v scan=%d field=%q", options, store.optionScanID, store.optionField)
	}

	if _, err := service.ListFilterOptionsByScan(context.Background(), 7, "url"); !errors.Is(err, ErrUnsupportedScreenshotSnapshotFilter) {
		t.Fatalf("expected unsupported filter option field, got %v", err)
	}
}
