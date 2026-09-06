package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type websiteCommandStoreStub struct {
	websiteByID              map[int]*assetdomain.Website
	batchCreateIn            []assetdomain.Website
	batchUpsertIn            []assetdomain.Website
	batchUpsertTechnologyIn  []assetdomain.WebsiteTechnology
	deletedID                int
	batchDeletedID           []int
	findByIDErr              error
	batchCreateErr           error
	deleteErr                error
	batchDeleteErr           error
	batchUpsertErr           error
	batchUpsertTechnologyErr error
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

func (stub *websiteCommandStoreStub) BatchCreateContext(_ context.Context, websites []assetdomain.Website) (int, error) {
	if stub.batchCreateErr != nil {
		return 0, stub.batchCreateErr
	}
	stub.batchCreateIn = append([]assetdomain.Website(nil), websites...)
	return len(websites), nil
}

func (stub *websiteCommandStoreStub) Delete(id int) error {
	if stub.deleteErr != nil {
		return stub.deleteErr
	}
	stub.deletedID = id
	return nil
}

func (stub *websiteCommandStoreStub) BatchDelete(ids []int) (int64, error) {
	if stub.batchDeleteErr != nil {
		return 0, stub.batchDeleteErr
	}
	stub.batchDeletedID = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

func (stub *websiteCommandStoreStub) BatchUpsertContext(_ context.Context, websites []assetdomain.Website) (int64, error) {
	if stub.batchUpsertErr != nil {
		return 0, stub.batchUpsertErr
	}
	stub.batchUpsertIn = append([]assetdomain.Website(nil), websites...)
	return int64(len(websites)), nil
}

func (stub *websiteCommandStoreStub) BatchUpsertTechnologyContext(_ context.Context, _ int, websites []assetdomain.WebsiteTechnology) (int64, error) {
	if stub.batchUpsertTechnologyErr != nil {
		return 0, stub.batchUpsertTechnologyErr
	}
	stub.batchUpsertTechnologyIn = append([]assetdomain.WebsiteTechnology(nil), websites...)
	return int64(len(websites)), nil
}

type websiteTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
	ctx     context.Context
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

func (stub *websiteTargetLookupStub) GetActiveByIDContext(ctx context.Context, id int) (*assetdomain.TargetRef, error) {
	stub.ctx = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.GetActiveByID(id)
}

func TestWebsiteCommandServiceBatchCreateAndDelete(t *testing.T) {
	store := &websiteCommandStoreStub{websiteByID: map[int]*assetdomain.Website{10: {ID: 10}}}
	lookup := &websiteTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewWebsiteCommandService(store, lookup)
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "caller")

	count, err := service.BatchCreate(ctx, 1, []string{"https://example.com", "https://foo.com"})
	if err != nil {
		t.Fatalf("batch create failed: %v", err)
	}
	if count != 1 || len(store.batchCreateIn) != 1 {
		t.Fatalf("unexpected create result count=%d size=%d", count, len(store.batchCreateIn))
	}
	if lookup.ctx != ctx || lookup.ctx.Value(contextKey{}) != "caller" {
		t.Fatal("target lookup did not preserve caller context")
	}

	if err := service.Delete(context.Background(), 10); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if store.deletedID != 10 {
		t.Fatalf("expected deleted id 10, got %d", store.deletedID)
	}
}

func TestWebsiteCommandServiceBatchUpsertAndErrors(t *testing.T) {
	store := &websiteCommandStoreStub{}
	lookup := &websiteTargetLookupStub{targets: map[int]*assetdomain.TargetRef{2: {ID: 2, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewWebsiteCommandService(store, lookup)

	count, err := service.BatchUpsert(context.Background(), 2, []WebsiteUpsertItem{
		{URL: "https://example.com", Host: "example.com", Title: "ok"},
		{URL: "https://bad.com", Host: "bad.com", Title: "no"},
	})
	if err != nil {
		t.Fatalf("batch upsert failed: %v", err)
	}
	if count != 1 || len(store.batchUpsertIn) != 1 {
		t.Fatalf("unexpected upsert result count=%d size=%d", count, len(store.batchUpsertIn))
	}

	lookup.err = gorm.ErrRecordNotFound
	_, err = service.BatchCreate(context.Background(), 99, []string{"https://x.com"})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}

func TestWebsiteCommandServiceBatchUpsertRejectsMissingOrConflictingHost(t *testing.T) {
	store := &websiteCommandStoreStub{}
	lookup := &websiteTargetLookupStub{targets: map[int]*assetdomain.TargetRef{
		2: {ID: 2, Name: "example.com", Type: assetdomain.TargetTypeDomain},
	}}
	service := NewWebsiteCommandService(store, lookup)

	for _, item := range []WebsiteUpsertItem{
		{URL: "https://example.com"},
		{URL: "https://example.com", Host: "other.example.com"},
	} {
		if _, err := service.BatchUpsert(context.Background(), 2, []WebsiteUpsertItem{item}); err == nil {
			t.Fatalf("BatchUpsert(%+v) unexpectedly succeeded", item)
		}
	}
	if len(store.batchUpsertIn) != 0 {
		t.Fatalf("invalid host assertion reached the store: %#v", store.batchUpsertIn)
	}
}

func TestWebsiteCommandServiceBatchUpsertTechnologyScopesWinsAndPreservesValues(t *testing.T) {
	store := &websiteCommandStoreStub{}
	lookup := &websiteTargetLookupStub{targets: map[int]*assetdomain.TargetRef{
		7: {ID: 7, Name: "example.com", Type: assetdomain.TargetTypeDomain},
	}}
	service := NewWebsiteCommandService(store, lookup)

	summary, err := service.BatchUpsertTechnologyContext(context.Background(), 7, []assetdomain.WebsiteTechnology{
		{URL: "https://example.com/keep", Tech: []string{"obsolete"}},
		{URL: "https://outside.example.net", Tech: []string{"ignored"}},
		{URL: "https://example.com/keep", Tech: []string{"zeta", "Beta", "zeta"}},
		{URL: "https://api.example.com", Tech: []string{}},
	})
	if err != nil {
		t.Fatalf("BatchUpsertTechnologyContext() error = %v", err)
	}
	if summary.ReceivedItems != 4 || summary.ScopeFilteredItems != 1 || summary.DuplicateItems != 1 || summary.AssetCount != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if len(store.batchUpsertTechnologyIn) != 2 {
		t.Fatalf("technology writes = %#v", store.batchUpsertTechnologyIn)
	}
	keep := store.batchUpsertTechnologyIn[0]
	if keep.URL != "https://example.com/keep" || keep.Host != "example.com" || len(keep.Tech) != 3 || keep.Tech[0] != "zeta" || keep.Tech[1] != "Beta" || keep.Tech[2] != "zeta" {
		t.Fatalf("last winner changed submitted technology values: %#v", keep)
	}
	zeroMatch := store.batchUpsertTechnologyIn[1]
	if zeroMatch.URL != "https://api.example.com" || zeroMatch.Host != "api.example.com" || zeroMatch.Tech == nil || len(zeroMatch.Tech) != 0 {
		t.Fatalf("trusted zero match was not preserved: %#v", zeroMatch)
	}
}
