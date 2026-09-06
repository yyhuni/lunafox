package application

import (
	"context"
	"errors"
	"testing"
)

type scanQueryStoreStub struct {
	lastPage     int
	lastPageSize int
	lastTargetID int
	lastStatus   string
	lastFilter   string
	lastOrderBy  string
}

func (stub *scanQueryStoreStub) List(page, pageSize int, targetID int, status, filter, orderBy string) ([]QueryScan, int64, error) {
	stub.lastPage = page
	stub.lastPageSize = pageSize
	stub.lastTargetID = targetID
	stub.lastStatus = status
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
	return []QueryScan{}, 0, nil
}

func (stub *scanQueryStoreStub) GetDetailByID(id int) (*QueryScan, error) {
	return nil, ErrScanNotFound
}

func (stub *scanQueryStoreStub) GetGlobalStatsSummary() (*QueryStatistics, error) {
	return &QueryStatistics{}, nil
}

func TestScanQueryServiceListScansNormalizesFilterAndCreatedAtOrderBy(t *testing.T) {
	store := &scanQueryStoreStub{}
	service := NewScanQueryService(store)

	_, _, err := service.ListScans(context.Background(), ScanListFilter{
		Page:     2,
		PageSize: 25,
		TargetID: 7,
		Filter:   `(status=="running" || status=="failed") && targetName="acme"`,
		OrderBy:  "createdAt",
	})
	if err != nil {
		t.Fatalf("ListScans failed: %v", err)
	}
	if store.lastPage != 2 || store.lastPageSize != 25 || store.lastTargetID != 7 {
		t.Fatalf("unexpected paging/target args: page=%d pageSize=%d target=%d", store.lastPage, store.lastPageSize, store.lastTargetID)
	}
	if store.lastFilter != `(status=="running" || status=="failed") && targetName="acme"` {
		t.Fatalf("unexpected filter %q", store.lastFilter)
	}
	if store.lastOrderBy != "createdAt asc" {
		t.Fatalf("expected createdAt asc orderBy, got %q", store.lastOrderBy)
	}
}

func TestScanQueryServiceListScansDefaultsToCreatedAtDescending(t *testing.T) {
	store := &scanQueryStoreStub{}
	service := NewScanQueryService(store)

	_, _, err := service.ListScans(context.Background(), ScanListFilter{})
	if err != nil {
		t.Fatalf("ListScans failed: %v", err)
	}
	if store.lastOrderBy != "createdAt desc" {
		t.Fatalf("expected default createdAt desc orderBy, got %q", store.lastOrderBy)
	}
}

func TestScanQueryServiceRejectsUnsupportedScanFilterAndOrderBy(t *testing.T) {
	service := NewScanQueryService(&scanQueryStoreStub{})

	if _, _, err := service.ListScans(context.Background(), ScanListFilter{Filter: `agentName="agent-1"`}); !errors.Is(err, ErrUnsupportedScanFilter) {
		t.Fatalf("expected ErrUnsupportedScanFilter, got %v", err)
	}
	if _, _, err := service.ListScans(context.Background(), ScanListFilter{OrderBy: "status desc"}); !errors.Is(err, ErrUnsupportedScanOrderBy) {
		t.Fatalf("expected ErrUnsupportedScanOrderBy, got %v", err)
	}
}
