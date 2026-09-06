package application

import (
	"context"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type subdomainSnapshotQueryStoreStub struct {
	items        []snapshotdomain.SubdomainSnapshot
	total        int64
	count        int64
	listErr      error
	forEachErr   error
	countErr     error
	visitErr     error
	listScanID   int
	listPage     int
	listPageSize int
	listFilter   string
	listOrderBy  string
}

func (stub *subdomainSnapshotQueryStoreStub) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.SubdomainSnapshot, int64, error) {
	stub.listScanID = scanID
	stub.listPage = page
	stub.listPageSize = pageSize
	stub.listFilter = filter
	stub.listOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]snapshotdomain.SubdomainSnapshot(nil), stub.items...), stub.total, nil
}

func (stub *subdomainSnapshotQueryStoreStub) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.SubdomainSnapshot) error) error {
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

func (stub *subdomainSnapshotQueryStoreStub) CountByScanID(scanID int) (int64, error) {
	_ = scanID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func TestSubdomainSnapshotQueryServiceListAndCount(t *testing.T) {
	store := &subdomainSnapshotQueryStoreStub{items: []snapshotdomain.SubdomainSnapshot{{ID: 1}}, total: 1, count: 4}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 3}}
	service := NewSubdomainSnapshotQueryService(store, lookup)

	result, err := service.ListByScan(context.Background(), 3, SubdomainSnapshotListQueryInput{
		PageSize: 20,
		Filter:   `dnsName="api"`,
		OrderBy:  "dnsName desc",
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(result.Subdomains) != 1 || result.TotalSize != 1 {
		t.Fatalf("unexpected list result len=%d total=%d", len(result.Subdomains), result.TotalSize)
	}
	if store.listScanID != 3 || store.listPage != 1 || store.listPageSize != 20 || store.listFilter != `dnsName="api"` || store.listOrderBy != "dnsName desc" {
		t.Fatalf("unexpected list args: %+v", store)
	}

	count, err := service.CountByScan(context.Background(), 3)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected count 4, got %d", count)
	}
}

func TestSubdomainSnapshotQueryServiceForEachByScan(t *testing.T) {
	store := &subdomainSnapshotQueryStoreStub{items: []snapshotdomain.SubdomainSnapshot{{ID: 1}, {ID: 2}}}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 3}}
	service := NewSubdomainSnapshotQueryService(store, lookup)

	var ids []int
	err := service.ForEachByScan(context.Background(), 3, func(item snapshotdomain.SubdomainSnapshot) error {
		ids = append(ids, item.ID)
		return nil
	})
	if err != nil {
		t.Fatalf("stream failed: %v", err)
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("unexpected streamed ids: %v", ids)
	}
}

func TestSubdomainSnapshotQueryServiceScanNotFound(t *testing.T) {
	store := &subdomainSnapshotQueryStoreStub{}
	lookup := &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound}
	service := NewSubdomainSnapshotQueryService(store, lookup)

	_, err := service.ListByScan(context.Background(), 1, SubdomainSnapshotListQueryInput{PageSize: 20})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}
}

func TestSubdomainSnapshotQueryServiceRejectsUnsupportedFilterOrderAndMismatchedToken(t *testing.T) {
	store := &subdomainSnapshotQueryStoreStub{}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 7}}
	service := NewSubdomainSnapshotQueryService(store, lookup)

	if _, err := service.ListByScan(context.Background(), 7, SubdomainSnapshotListQueryInput{Filter: `name="api"`}); !errors.Is(err, ErrUnsupportedSubdomainSnapshotFilter) {
		t.Fatalf("expected unsupported filter, got %v", err)
	}
	if store.listScanID != 0 {
		t.Fatal("unsupported filter must fail before store access")
	}

	if _, err := service.ListByScan(context.Background(), 7, SubdomainSnapshotListQueryInput{OrderBy: "targetCount desc"}); !errors.Is(err, ErrUnsupportedSubdomainSnapshotOrderBy) {
		t.Fatalf("expected unsupported orderBy, got %v", err)
	}

	token, err := encodeSubdomainSnapshotListPageToken(subdomainSnapshotListPageTokenPayload{
		Version:  subdomainSnapshotListTokenVersion,
		Page:     2,
		PageSize: 10,
		ScanID:   8,
		Filter:   `dnsName="api"`,
		OrderBy:  "createdAt desc",
	})
	if err != nil {
		t.Fatalf("encode token: %v", err)
	}
	if _, err := service.ListByScan(context.Background(), 7, SubdomainSnapshotListQueryInput{PageSize: 10, PageToken: token, Filter: `dnsName="api"`}); !errors.Is(err, ErrInvalidSubdomainSnapshotPageToken) {
		t.Fatalf("expected scan-bound token mismatch, got %v", err)
	}
}
