package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type endpointCommandStoreStub struct {
	endpointByID   map[int]*assetdomain.Endpoint
	bulkCreateIn   []assetdomain.Endpoint
	bulkUpsertIn   []assetdomain.Endpoint
	deletedID      int
	bulkDeletedIDs []int
	findByIDErr    error
	bulkCreateErr  error
	deleteErr      error
	bulkDeleteErr  error
	bulkUpsertErr  error
}

func (stub *endpointCommandStoreStub) GetByID(id int) (*assetdomain.Endpoint, error) {
	if stub.findByIDErr != nil {
		return nil, stub.findByIDErr
	}
	item, ok := stub.endpointByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyItem := *item
	return &copyItem, nil
}

func (stub *endpointCommandStoreStub) BulkCreate(endpoints []assetdomain.Endpoint) (int, error) {
	if stub.bulkCreateErr != nil {
		return 0, stub.bulkCreateErr
	}
	stub.bulkCreateIn = append([]assetdomain.Endpoint(nil), endpoints...)
	return len(endpoints), nil
}

func (stub *endpointCommandStoreStub) Delete(id int) error {
	if stub.deleteErr != nil {
		return stub.deleteErr
	}
	stub.deletedID = id
	return nil
}

func (stub *endpointCommandStoreStub) BulkDelete(ids []int) (int64, error) {
	if stub.bulkDeleteErr != nil {
		return 0, stub.bulkDeleteErr
	}
	stub.bulkDeletedIDs = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

func (stub *endpointCommandStoreStub) BulkUpsert(endpoints []assetdomain.Endpoint) (int64, error) {
	if stub.bulkUpsertErr != nil {
		return 0, stub.bulkUpsertErr
	}
	stub.bulkUpsertIn = append([]assetdomain.Endpoint(nil), endpoints...)
	return int64(len(endpoints)), nil
}

type endpointTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
}

func (stub *endpointTargetLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
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

func TestEndpointCommandServiceBulkCreateAndUpsert(t *testing.T) {
	store := &endpointCommandStoreStub{endpointByID: map[int]*assetdomain.Endpoint{5: {ID: 5}}}
	lookup := &endpointTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewEndpointCommandService(store, lookup)

	created, err := service.BulkCreate(context.Background(), 1, []string{"https://example.com/a", "https://foo.com"})
	if err != nil {
		t.Fatalf("bulk create failed: %v", err)
	}
	if created != 1 || len(store.bulkCreateIn) != 1 {
		t.Fatalf("unexpected create result created=%d size=%d", created, len(store.bulkCreateIn))
	}

	affected, err := service.BulkUpsert(context.Background(), 1, []EndpointUpsertItem{
		{URL: "https://example.com/x"},
		{URL: "https://foo.com/x"},
	})
	if err != nil {
		t.Fatalf("bulk upsert failed: %v", err)
	}
	if affected != 1 || len(store.bulkUpsertIn) != 1 {
		t.Fatalf("unexpected upsert result affected=%d size=%d", affected, len(store.bulkUpsertIn))
	}
}

func TestEndpointCommandServiceDeleteAndErrors(t *testing.T) {
	store := &endpointCommandStoreStub{endpointByID: map[int]*assetdomain.Endpoint{3: {ID: 3}}}
	lookup := &endpointTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewEndpointCommandService(store, lookup)

	if err := service.Delete(context.Background(), 3); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if store.deletedID != 3 {
		t.Fatalf("expected deleted id 3, got %d", store.deletedID)
	}

	lookup.err = gorm.ErrRecordNotFound
	_, err := service.BulkCreate(context.Background(), 9, []string{"https://a.com"})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}

	store.endpointByID = map[int]*assetdomain.Endpoint{}
	err = service.Delete(context.Background(), 99)
	if !errors.Is(err, ErrEndpointNotFound) {
		t.Fatalf("expected ErrEndpointNotFound, got %v", err)
	}
}
