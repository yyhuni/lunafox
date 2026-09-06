package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type subdomainCommandStoreStub struct {
	batchCreateIn   []assetdomain.Subdomain
	batchDeletedIDs []int
	batchCreateErr  error
	batchDeleteErr  error
}

func (stub *subdomainCommandStoreStub) BatchCreateContext(_ context.Context, items []assetdomain.Subdomain) (int, error) {
	if stub.batchCreateErr != nil {
		return 0, stub.batchCreateErr
	}
	stub.batchCreateIn = append([]assetdomain.Subdomain(nil), items...)
	return len(items), nil
}

func (stub *subdomainCommandStoreStub) BatchDelete(ids []int) (int64, error) {
	if stub.batchDeleteErr != nil {
		return 0, stub.batchDeleteErr
	}
	stub.batchDeletedIDs = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

type subdomainTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
	ctx     context.Context
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

func (stub *subdomainTargetLookupStub) GetActiveByIDContext(ctx context.Context, id int) (*assetdomain.TargetRef, error) {
	stub.ctx = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.GetActiveByID(id)
}

func TestSubdomainCommandServiceBatchCreateAndDelete(t *testing.T) {
	store := &subdomainCommandStoreStub{}
	lookup := &subdomainTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewSubdomainCommandService(store, lookup)
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "caller")

	created, err := service.BatchCreate(ctx, 1, []string{"api.example.com", "foo.com"})
	if err != nil {
		t.Fatalf("batch create failed: %v", err)
	}
	if created != 1 || len(store.batchCreateIn) != 1 {
		t.Fatalf("unexpected create result created=%d size=%d", created, len(store.batchCreateIn))
	}
	if lookup.ctx != ctx || lookup.ctx.Value(contextKey{}) != "caller" {
		t.Fatal("target lookup did not preserve caller context")
	}

	deleted, err := service.BatchDelete(context.Background(), []int{1, 2, 3})
	if err != nil {
		t.Fatalf("batch delete failed: %v", err)
	}
	if deleted != 3 || len(store.batchDeletedIDs) != 3 {
		t.Fatalf("unexpected batch delete result deleted=%d ids=%v", deleted, store.batchDeletedIDs)
	}
}

func TestSubdomainCommandServiceRejectsNonCanonicalWithoutRepair(t *testing.T) {
	store := &subdomainCommandStoreStub{}
	lookup := &subdomainTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewSubdomainCommandService(store, lookup)

	if _, err := service.BatchCreate(context.Background(), 1, []string{"Api.Example.COM.", "api.example.com"}); err == nil {
		t.Fatal("BatchCreate silently repaired a non-canonical DNS name")
	}
	if len(store.batchCreateIn) != 0 {
		t.Fatalf("invalid DNS name reached the store: %+v", store.batchCreateIn)
	}
}

func TestSubdomainCommandServiceErrors(t *testing.T) {
	store := &subdomainCommandStoreStub{}
	lookup := &subdomainTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeIP}}}
	service := NewSubdomainCommandService(store, lookup)

	_, err := service.BatchCreate(context.Background(), 1, []string{"api.example.com"})
	if !errors.Is(err, ErrSubdomainInvalidTargetType) {
		t.Fatalf("expected ErrSubdomainInvalidTargetType, got %v", err)
	}

	lookup.err = gorm.ErrRecordNotFound
	_, err = service.BatchCreate(context.Background(), 9, []string{"api.example.com"})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}
