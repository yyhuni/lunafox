package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type websiteQueryStoreStub struct {
	items       []assetdomain.Website
	total       int64
	count       int64
	findErr     error
	countErr    error
	streamErr   error
	scannedItem *assetdomain.Website
}

func (stub *websiteQueryStoreStub) FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Website, int64, error) {
	_ = targetID
	_ = page
	_ = pageSize
	_ = filter
	if stub.findErr != nil {
		return nil, 0, stub.findErr
	}
	return append([]assetdomain.Website(nil), stub.items...), stub.total, nil
}

func (stub *websiteQueryStoreStub) StreamByTargetID(targetID int) (*sql.Rows, error) {
	_ = targetID
	if stub.streamErr != nil {
		return nil, stub.streamErr
	}
	return &sql.Rows{}, nil
}

func (stub *websiteQueryStoreStub) CountByTargetID(targetID int) (int64, error) {
	_ = targetID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func (stub *websiteQueryStoreStub) ScanRow(rows *sql.Rows) (*assetdomain.Website, error) {
	_ = rows
	if stub.scannedItem == nil {
		return &assetdomain.Website{}, nil
	}
	copyItem := *stub.scannedItem
	return &copyItem, nil
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
	service := NewWebsiteQueryService(store, lookup)

	items, total, err := service.ListByTarget(context.Background(), 7, 1, 20, "")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 || total != 1 {
		t.Fatalf("unexpected list result len=%d total=%d", len(items), total)
	}

	count, err := service.CountByTarget(context.Background(), 7)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}
}

func TestWebsiteQueryServiceTargetNotFound(t *testing.T) {
	store := &websiteQueryStoreStub{}
	lookup := &websiteTargetLookupQueryStub{err: gorm.ErrRecordNotFound}
	service := NewWebsiteQueryService(store, lookup)

	_, err := service.StreamByTarget(context.Background(), 1)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}
