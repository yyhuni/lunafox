package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type endpointQueryStoreStub struct {
	items      []assetdomain.Endpoint
	total      int64
	count      int64
	itemByID   map[int]*assetdomain.Endpoint
	listErr    error
	findErr    error
	streamErr  error
	countErr   error
	scannedErr error
}

func (stub *endpointQueryStoreStub) FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Endpoint, int64, error) {
	_ = targetID
	_ = page
	_ = pageSize
	_ = filter
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]assetdomain.Endpoint(nil), stub.items...), stub.total, nil
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

func (stub *endpointQueryStoreStub) StreamByTargetID(targetID int) (*sql.Rows, error) {
	_ = targetID
	if stub.streamErr != nil {
		return nil, stub.streamErr
	}
	return &sql.Rows{}, nil
}

func (stub *endpointQueryStoreStub) CountByTargetID(targetID int) (int64, error) {
	_ = targetID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func (stub *endpointQueryStoreStub) ScanRow(rows *sql.Rows) (*assetdomain.Endpoint, error) {
	_ = rows
	if stub.scannedErr != nil {
		return nil, stub.scannedErr
	}
	return &assetdomain.Endpoint{ID: 10}, nil
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

	items, total, err := service.ListByTarget(context.Background(), 7, 1, 20, "")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 2 || total != 2 {
		t.Fatalf("unexpected list result len=%d total=%d", len(items), total)
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

	_, _, err := service.ListByTarget(context.Background(), 1, 1, 20, "")
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}
