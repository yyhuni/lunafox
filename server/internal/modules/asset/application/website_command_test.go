package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type websiteCommandStoreStub struct {
	websiteByID   map[int]*assetdomain.Website
	bulkCreateIn  []assetdomain.Website
	bulkUpsertIn  []assetdomain.Website
	deletedID     int
	bulkDeletedID []int
	findByIDErr   error
	bulkCreateErr error
	deleteErr     error
	bulkDeleteErr error
	bulkUpsertErr error
}

func (stub *websiteCommandStoreStub) GetByID(id int) (*assetdomain.Website, error) {
	if stub.findByIDErr != nil {
		return nil, stub.findByIDErr
	}
	item, ok := stub.websiteByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyItem := *item
	return &copyItem, nil
}

func (stub *websiteCommandStoreStub) BulkCreate(websites []assetdomain.Website) (int, error) {
	if stub.bulkCreateErr != nil {
		return 0, stub.bulkCreateErr
	}
	stub.bulkCreateIn = append([]assetdomain.Website(nil), websites...)
	return len(websites), nil
}

func (stub *websiteCommandStoreStub) Delete(id int) error {
	if stub.deleteErr != nil {
		return stub.deleteErr
	}
	stub.deletedID = id
	return nil
}

func (stub *websiteCommandStoreStub) BulkDelete(ids []int) (int64, error) {
	if stub.bulkDeleteErr != nil {
		return 0, stub.bulkDeleteErr
	}
	stub.bulkDeletedID = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

func (stub *websiteCommandStoreStub) BulkUpsert(websites []assetdomain.Website) (int64, error) {
	if stub.bulkUpsertErr != nil {
		return 0, stub.bulkUpsertErr
	}
	stub.bulkUpsertIn = append([]assetdomain.Website(nil), websites...)
	return int64(len(websites)), nil
}

type websiteTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
}

func (stub *websiteTargetLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
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

func TestWebsiteCommandServiceBulkCreateAndDelete(t *testing.T) {
	store := &websiteCommandStoreStub{websiteByID: map[int]*assetdomain.Website{10: {ID: 10}}}
	lookup := &websiteTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewWebsiteCommandService(store, lookup)

	count, err := service.BulkCreate(context.Background(), 1, []string{"https://example.com", "https://foo.com"})
	if err != nil {
		t.Fatalf("bulk create failed: %v", err)
	}
	if count != 1 || len(store.bulkCreateIn) != 1 {
		t.Fatalf("unexpected create result count=%d size=%d", count, len(store.bulkCreateIn))
	}

	if err := service.Delete(context.Background(), 10); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if store.deletedID != 10 {
		t.Fatalf("expected deleted id 10, got %d", store.deletedID)
	}
}

func TestWebsiteCommandServiceBulkUpsertAndErrors(t *testing.T) {
	store := &websiteCommandStoreStub{}
	lookup := &websiteTargetLookupStub{targets: map[int]*assetdomain.TargetRef{2: {ID: 2, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewWebsiteCommandService(store, lookup)

	count, err := service.BulkUpsert(context.Background(), 2, []WebsiteUpsertItem{
		{URL: "https://example.com", Title: "ok"},
		{URL: "https://bad.com", Title: "no"},
	})
	if err != nil {
		t.Fatalf("bulk upsert failed: %v", err)
	}
	if count != 1 || len(store.bulkUpsertIn) != 1 {
		t.Fatalf("unexpected upsert result count=%d size=%d", count, len(store.bulkUpsertIn))
	}

	lookup.err = gorm.ErrRecordNotFound
	_, err = service.BulkCreate(context.Background(), 99, []string{"https://x.com"})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}
