package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type subdomainCommandStoreStub struct {
	bulkCreateIn   []assetdomain.Subdomain
	bulkDeletedIDs []int
	bulkCreateErr  error
	bulkDeleteErr  error
}

func (stub *subdomainCommandStoreStub) BulkCreate(items []assetdomain.Subdomain) (int, error) {
	if stub.bulkCreateErr != nil {
		return 0, stub.bulkCreateErr
	}
	stub.bulkCreateIn = append([]assetdomain.Subdomain(nil), items...)
	return len(items), nil
}

func (stub *subdomainCommandStoreStub) BulkDelete(ids []int) (int64, error) {
	if stub.bulkDeleteErr != nil {
		return 0, stub.bulkDeleteErr
	}
	stub.bulkDeletedIDs = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

type subdomainTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
}

func (stub *subdomainTargetLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
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

func TestSubdomainCommandServiceBulkCreateAndDelete(t *testing.T) {
	store := &subdomainCommandStoreStub{}
	lookup := &subdomainTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewSubdomainCommandService(store, lookup)

	created, err := service.BulkCreate(context.Background(), 1, []string{"api.example.com", "foo.com"})
	if err != nil {
		t.Fatalf("bulk create failed: %v", err)
	}
	if created != 1 || len(store.bulkCreateIn) != 1 {
		t.Fatalf("unexpected create result created=%d size=%d", created, len(store.bulkCreateIn))
	}

	deleted, err := service.BulkDelete(context.Background(), []int{1, 2, 3})
	if err != nil {
		t.Fatalf("bulk delete failed: %v", err)
	}
	if deleted != 3 || len(store.bulkDeletedIDs) != 3 {
		t.Fatalf("unexpected bulk delete result deleted=%d ids=%v", deleted, store.bulkDeletedIDs)
	}
}

func TestSubdomainCommandServiceErrors(t *testing.T) {
	store := &subdomainCommandStoreStub{}
	lookup := &subdomainTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeIP}}}
	service := NewSubdomainCommandService(store, lookup)

	_, err := service.BulkCreate(context.Background(), 1, []string{"api.example.com"})
	if !errors.Is(err, ErrSubdomainInvalidTargetType) {
		t.Fatalf("expected ErrSubdomainInvalidTargetType, got %v", err)
	}

	lookup.err = gorm.ErrRecordNotFound
	_, err = service.BulkCreate(context.Background(), 9, []string{"api.example.com"})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}
