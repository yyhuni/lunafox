package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type endpointCommandStoreStub struct {
	endpointByID     map[int]*assetdomain.Endpoint
	batchCreateIn    []assetdomain.Endpoint
	batchUpsertIn    []assetdomain.Endpoint
	deletedID        int
	batchDeletedIDs  []int
	batchCreateCalls int
	batchUpsertCalls int
	findByIDErr      error
	batchCreateErr   error
	deleteErr        error
	batchDeleteErr   error
	batchUpsertErr   error
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

func (stub *endpointCommandStoreStub) BatchCreate(endpoints []assetdomain.Endpoint) (int, error) {
	stub.batchCreateCalls++
	if stub.batchCreateErr != nil {
		return 0, stub.batchCreateErr
	}
	stub.batchCreateIn = append([]assetdomain.Endpoint(nil), endpoints...)
	return len(endpoints), nil
}

func (stub *endpointCommandStoreStub) Delete(id int) error {
	if stub.deleteErr != nil {
		return stub.deleteErr
	}
	stub.deletedID = id
	return nil
}

func (stub *endpointCommandStoreStub) BatchDelete(ids []int) (int64, error) {
	if stub.batchDeleteErr != nil {
		return 0, stub.batchDeleteErr
	}
	stub.batchDeletedIDs = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

func (stub *endpointCommandStoreStub) BatchUpsert(endpoints []assetdomain.Endpoint) (int64, error) {
	stub.batchUpsertCalls++
	if stub.batchUpsertErr != nil {
		return 0, stub.batchUpsertErr
	}
	stub.batchUpsertIn = append([]assetdomain.Endpoint(nil), endpoints...)
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

func TestEndpointCommandServiceBatchCreateAndUpsert(t *testing.T) {
	store := &endpointCommandStoreStub{endpointByID: map[int]*assetdomain.Endpoint{5: {ID: 5}}}
	lookup := &endpointTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewEndpointCommandService(store, lookup)

	created, err := service.BatchCreate(context.Background(), 1, []string{"https://example.com/a", "https://foo.com", "https://api.example.com:8443/health"})
	if err != nil {
		t.Fatalf("batch create failed: %v", err)
	}
	if created != 2 || len(store.batchCreateIn) != 2 {
		t.Fatalf("unexpected create result created=%d size=%d", created, len(store.batchCreateIn))
	}
	if store.batchCreateIn[0].TargetID != 1 || store.batchCreateIn[0].URL != "https://example.com/a" || store.batchCreateIn[0].Host != "example.com" {
		t.Fatalf("unexpected first created endpoint: %+v", store.batchCreateIn[0])
	}
	if store.batchCreateIn[1].TargetID != 1 || store.batchCreateIn[1].URL != "https://api.example.com:8443/health" || store.batchCreateIn[1].Host != "api.example.com" {
		t.Fatalf("unexpected second created endpoint: %+v", store.batchCreateIn[1])
	}

	affected, err := service.BatchUpsert(context.Background(), 1, []EndpointUpsertItem{
		{URL: "https://example.com/x", Host: "example.com"},
		{URL: "https://foo.com/x", Host: "foo.com"},
	})
	if err != nil {
		t.Fatalf("batch upsert failed: %v", err)
	}
	if affected != 1 || len(store.batchUpsertIn) != 1 {
		t.Fatalf("unexpected upsert result affected=%d size=%d", affected, len(store.batchUpsertIn))
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
	_, err := service.BatchCreate(context.Background(), 9, []string{"https://a.com"})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}

	store.endpointByID = map[int]*assetdomain.Endpoint{}
	store.deletedID = 0
	err = service.Delete(context.Background(), 99)
	if !errors.Is(err, ErrEndpointNotFound) {
		t.Fatalf("expected ErrEndpointNotFound, got %v", err)
	}
	if store.deletedID != 0 {
		t.Fatalf("expected delete to short-circuit on missing endpoint, got deleted id %d", store.deletedID)
	}
}

func TestEndpointCommandServiceBatchUpsertMapsHostsAndFields(t *testing.T) {
	statusCode := 200
	contentLength := 1024
	vhost := true

	store := &endpointCommandStoreStub{}
	lookup := &endpointTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewEndpointCommandService(store, lookup)

	affected, err := service.BatchUpsert(context.Background(), 1, []EndpointUpsertItem{
		{
			URL:             "https://example.com/login",
			Host:            "example.com",
			Location:        "/signin",
			Title:           "Login",
			Webserver:       "nginx",
			ContentType:     "text/html",
			StatusCode:      &statusCode,
			ContentLength:   &contentLength,
			ResponseBody:    "ok",
			Tech:            []string{"go", "htmx"},
			Vhost:           &vhost,
			ResponseHeaders: "server: nginx",
		},
		{
			URL:  "https://api.example.com:8443/health",
			Host: "api.example.com",
		},
		{
			URL:  "https://off-target.com/skip",
			Host: "off-target.com",
		},
	})
	if err != nil {
		t.Fatalf("batch upsert failed: %v", err)
	}
	if affected != 2 || len(store.batchUpsertIn) != 2 {
		t.Fatalf("unexpected upsert result affected=%d size=%d", affected, len(store.batchUpsertIn))
	}

	first := store.batchUpsertIn[0]
	if first.Host != "example.com" || first.Location != "/signin" || first.Title != "Login" || first.Webserver != "nginx" {
		t.Fatalf("expected explicit fields to be preserved, got %+v", first)
	}
	if first.StatusCode == nil || *first.StatusCode != statusCode || first.ContentLength == nil || *first.ContentLength != contentLength || first.ResponseBody != "ok" || first.ResponseHeaders != "server: nginx" {
		t.Fatalf("expected response fields to be preserved, got %+v", first)
	}
	if len(first.Tech) != 2 || first.Tech[0] != "go" || first.Tech[1] != "htmx" || first.Vhost == nil || *first.Vhost != vhost {
		t.Fatalf("expected tech and vhost to be preserved, got %+v", first)
	}

	second := store.batchUpsertIn[1]
	if second.Host != "api.example.com" {
		t.Fatalf("expected matching host assertion to be preserved, got %+v", second)
	}
}

func TestEndpointCommandServiceBatchUpsertRejectsMissingOrConflictingHost(t *testing.T) {
	store := &endpointCommandStoreStub{}
	lookup := &endpointTargetLookupStub{targets: map[int]*assetdomain.TargetRef{
		1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
	}}
	service := NewEndpointCommandService(store, lookup)

	for _, item := range []EndpointUpsertItem{
		{URL: "https://example.com"},
		{URL: "https://example.com", Host: "other.example.com"},
	} {
		if _, err := service.BatchUpsert(context.Background(), 1, []EndpointUpsertItem{item}); err == nil {
			t.Fatalf("BatchUpsert(%+v) unexpectedly succeeded", item)
		}
	}
	if len(store.batchUpsertIn) != 0 {
		t.Fatalf("invalid host assertion reached the store: %#v", store.batchUpsertIn)
	}
}

func TestEndpointCommandServiceDeletePassesThroughStoreErrors(t *testing.T) {
	service := NewEndpointCommandService(
		&endpointCommandStoreStub{findByIDErr: errors.New("lookup failed")},
		&endpointTargetLookupStub{},
	)

	if err := service.Delete(context.Background(), 1); err == nil || err.Error() != "lookup failed" {
		t.Fatalf("expected get by id error, got %v", err)
	}

	store := &endpointCommandStoreStub{
		endpointByID: map[int]*assetdomain.Endpoint{2: {ID: 2}},
		deleteErr:    errors.New("delete failed"),
	}
	service = NewEndpointCommandService(store, &endpointTargetLookupStub{})
	if err := service.Delete(context.Background(), 2); err == nil || err.Error() != "delete failed" {
		t.Fatalf("expected delete error, got %v", err)
	}
}

func TestEndpointCommandServiceBatchCreateStoreErrorAndShortCircuit(t *testing.T) {
	t.Run("store error propagates", func(t *testing.T) {
		wantErr := errors.New("batch create failed")
		store := &endpointCommandStoreStub{batchCreateErr: wantErr}
		lookup := &endpointTargetLookupStub{
			targets: map[int]*assetdomain.TargetRef{
				1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
			},
		}
		service := NewEndpointCommandService(store, lookup)

		created, err := service.BatchCreate(context.Background(), 1, []string{"https://example.com/a"})
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected store error %v, got created=%d err=%v", wantErr, created, err)
		}
	})

	t.Run("all filtered skips store", func(t *testing.T) {
		store := &endpointCommandStoreStub{batchCreateErr: errors.New("should not be called")}
		lookup := &endpointTargetLookupStub{
			targets: map[int]*assetdomain.TargetRef{
				1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
			},
		}
		service := NewEndpointCommandService(store, lookup)

		created, err := service.BatchCreate(context.Background(), 1, []string{"https://off-target.com/a"})
		if err != nil {
			t.Fatalf("expected filtered batch create to short-circuit, got created=%d err=%v", created, err)
		}
		if created != 0 {
			t.Fatalf("expected created count 0, got %d", created)
		}
		if store.batchCreateCalls != 0 {
			t.Fatalf("expected store batch create to be skipped, got %d calls", store.batchCreateCalls)
		}
	})
}

func TestEndpointCommandServiceBatchUpsertPassesThroughLookupErrors(t *testing.T) {
	store := &endpointCommandStoreStub{}
	service := NewEndpointCommandService(
		store,
		&endpointTargetLookupStub{err: errors.New("lookup failed")},
	)

	affected, err := service.BatchUpsert(context.Background(), 1, []EndpointUpsertItem{{URL: "https://example.com/a", Host: "example.com"}})
	if err == nil || err.Error() != "lookup failed" {
		t.Fatalf("expected lookup error, got affected=%d err=%v", affected, err)
	}
	if store.batchUpsertCalls != 0 {
		t.Fatalf("expected store batch upsert to be skipped, got %d calls", store.batchUpsertCalls)
	}
}

func TestEndpointCommandServiceBatchCreatePassesThroughLookupErrors(t *testing.T) {
	wantErr := errors.New("lookup failed")
	store := &endpointCommandStoreStub{}
	service := NewEndpointCommandService(
		store,
		&endpointTargetLookupStub{err: wantErr},
	)

	created, err := service.BatchCreate(context.Background(), 1, []string{"https://example.com/a"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected lookup error %v, got created=%d err=%v", wantErr, created, err)
	}
	if store.batchCreateCalls != 0 {
		t.Fatalf("expected store batch create to be skipped, got %d calls", store.batchCreateCalls)
	}
}

func TestEndpointCommandServiceBatchUpsertStoreErrorAndShortCircuit(t *testing.T) {
	t.Run("store error propagates", func(t *testing.T) {
		wantErr := errors.New("batch upsert failed")
		store := &endpointCommandStoreStub{batchUpsertErr: wantErr}
		lookup := &endpointTargetLookupStub{
			targets: map[int]*assetdomain.TargetRef{
				1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
			},
		}
		service := NewEndpointCommandService(store, lookup)

		affected, err := service.BatchUpsert(context.Background(), 1, []EndpointUpsertItem{{URL: "https://example.com/a", Host: "example.com"}})
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected store error %v, got affected=%d err=%v", wantErr, affected, err)
		}
	})

	t.Run("all filtered skips store", func(t *testing.T) {
		store := &endpointCommandStoreStub{batchUpsertErr: errors.New("should not be called")}
		lookup := &endpointTargetLookupStub{
			targets: map[int]*assetdomain.TargetRef{
				1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
			},
		}
		service := NewEndpointCommandService(store, lookup)

		affected, err := service.BatchUpsert(context.Background(), 1, []EndpointUpsertItem{{URL: "https://off-target.com/a", Host: "off-target.com"}})
		if err != nil {
			t.Fatalf("expected filtered batch upsert to short-circuit, got affected=%d err=%v", affected, err)
		}
		if affected != 0 {
			t.Fatalf("expected affected count 0, got %d", affected)
		}
		if store.batchUpsertCalls != 0 {
			t.Fatalf("expected store batch upsert to be skipped, got %d calls", store.batchUpsertCalls)
		}
	})
}
