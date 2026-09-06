package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type directoryCommandStoreStub struct {
	batchCreateIn    []assetdomain.Directory
	batchUpsertIn    []assetdomain.Directory
	batchDeletedIDs  []int
	batchCreateCalls int
	batchUpsertCalls int
	batchCreateErr   error
	batchDeleteErr   error
	batchUpsertErr   error
}

func (stub *directoryCommandStoreStub) BatchCreate(items []assetdomain.Directory) (int, error) {
	stub.batchCreateCalls++
	if stub.batchCreateErr != nil {
		return 0, stub.batchCreateErr
	}
	stub.batchCreateIn = append([]assetdomain.Directory(nil), items...)
	return len(items), nil
}

func (stub *directoryCommandStoreStub) BatchDelete(ids []int) (int64, error) {
	if stub.batchDeleteErr != nil {
		return 0, stub.batchDeleteErr
	}
	stub.batchDeletedIDs = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

func (stub *directoryCommandStoreStub) BatchUpsert(items []assetdomain.Directory) (int64, error) {
	stub.batchUpsertCalls++
	if stub.batchUpsertErr != nil {
		return 0, stub.batchUpsertErr
	}
	stub.batchUpsertIn = append([]assetdomain.Directory(nil), items...)
	return int64(len(items)), nil
}

func (stub *directoryCommandStoreStub) BatchUpsertContext(ctx context.Context, items []assetdomain.Directory) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return stub.BatchUpsert(items)
}

type directoryTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
}

func (stub *directoryTargetLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
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

func (stub *directoryTargetLookupStub) GetActiveByIDContext(ctx context.Context, id int) (*assetdomain.TargetRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.GetActiveByID(id)
}

func TestDirectoryCommandServiceBatchCreateAndUpsert(t *testing.T) {
	store := &directoryCommandStoreStub{}
	lookup := &directoryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewDirectoryCommandService(store, lookup)

	created, err := service.BatchCreate(context.Background(), 1, []string{"https://example.com/a", "https://foo.com", "https://example.com/b"})
	if err != nil {
		t.Fatalf("batch create failed: %v", err)
	}
	if created != 2 || len(store.batchCreateIn) != 2 {
		t.Fatalf("unexpected create result created=%d size=%d", created, len(store.batchCreateIn))
	}
	if store.batchCreateIn[0].TargetID != 1 || store.batchCreateIn[0].URL != "https://example.com/a" {
		t.Fatalf("unexpected first created directory: %+v", store.batchCreateIn[0])
	}
	if store.batchCreateIn[1].TargetID != 1 || store.batchCreateIn[1].URL != "https://example.com/b" {
		t.Fatalf("unexpected second created directory: %+v", store.batchCreateIn[1])
	}

	affected, err := service.BatchUpsert(context.Background(), 1, []DirectoryUpsertItem{{URL: "https://example.com/x"}, {URL: "https://foo.com/x"}})
	if err != nil {
		t.Fatalf("batch upsert failed: %v", err)
	}
	if affected != 1 || len(store.batchUpsertIn) != 1 {
		t.Fatalf("unexpected upsert result affected=%d size=%d", affected, len(store.batchUpsertIn))
	}
}

func TestDirectoryCommandServiceErrorsAndBatchDelete(t *testing.T) {
	store := &directoryCommandStoreStub{}
	lookup := &directoryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewDirectoryCommandService(store, lookup)

	lookup.err = gorm.ErrRecordNotFound
	_, err := service.BatchCreate(context.Background(), 9, []string{"https://a.com"})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}

	lookup.err = nil
	deleted, err := service.BatchDelete(context.Background(), []int{1, 2, 3})
	if err != nil {
		t.Fatalf("batch delete failed: %v", err)
	}
	if deleted != 3 || len(store.batchDeletedIDs) != 3 {
		t.Fatalf("unexpected batch delete result deleted=%d ids=%v", deleted, store.batchDeletedIDs)
	}
}

func TestDirectoryCommandServiceBatchUpsertMapsFieldsAndShortCircuits(t *testing.T) {
	status := 200
	contentLength := int64(512)
	duration := int64(37)

	store := &directoryCommandStoreStub{}
	lookup := &directoryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewDirectoryCommandService(store, lookup)

	affected, err := service.BatchUpsert(context.Background(), 1, []DirectoryUpsertItem{
		{
			URL:           "https://example.com/assets",
			Status:        &status,
			ContentLength: &contentLength,
			ContentType:   "text/html",
			Duration:      &duration,
		},
		{URL: "https://off-target.com/assets"},
	})
	if err != nil {
		t.Fatalf("batch upsert failed: %v", err)
	}
	if affected != 1 || len(store.batchUpsertIn) != 1 {
		t.Fatalf("unexpected upsert result affected=%d size=%d", affected, len(store.batchUpsertIn))
	}

	got := store.batchUpsertIn[0]
	if got.TargetID != 1 || got.URL != "https://example.com/assets" {
		t.Fatalf("unexpected upserted directory identity: %+v", got)
	}
	if got.Status == nil || *got.Status != status || got.ContentLength == nil || *got.ContentLength != contentLength || got.Duration == nil || *got.Duration != duration || got.ContentType != "text/html" {
		t.Fatalf("expected metadata fields to be preserved, got %+v", got)
	}

	store.batchUpsertIn = nil
	store.batchUpsertCalls = 0
	affected, err = service.BatchUpsert(context.Background(), 1, []DirectoryUpsertItem{{URL: "https://off-target.com/only"}})
	if err != nil {
		t.Fatalf("batch upsert with filtered items failed: %v", err)
	}
	if affected != 0 {
		t.Fatalf("expected no affected rows, got %d", affected)
	}
	if store.batchUpsertCalls != 0 {
		t.Fatalf("expected store batch upsert to be skipped, got %d calls", store.batchUpsertCalls)
	}
}

func TestDirectoryCommandServiceBatchCreatePassesThroughLookupErrors(t *testing.T) {
	store := &directoryCommandStoreStub{}
	service := NewDirectoryCommandService(store, &directoryTargetLookupStub{err: errors.New("lookup failed")})

	created, err := service.BatchCreate(context.Background(), 1, []string{"https://example.com/a"})
	if err == nil || err.Error() != "lookup failed" {
		t.Fatalf("expected lookup error, got created=%d err=%v", created, err)
	}
	if store.batchCreateCalls != 0 {
		t.Fatalf("expected store batch create to be skipped, got %d calls", store.batchCreateCalls)
	}
}

func TestDirectoryCommandServiceBatchCreateSkipsStoreWhenAllURLsFiltered(t *testing.T) {
	store := &directoryCommandStoreStub{batchCreateErr: errors.New("should not be called")}
	lookup := &directoryTargetLookupStub{
		targets: map[int]*assetdomain.TargetRef{
			1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
		},
	}
	service := NewDirectoryCommandService(store, lookup)

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
}

func TestDirectoryCommandServiceBatchUpsertLookupErrors(t *testing.T) {
	t.Run("target not found maps to module error", func(t *testing.T) {
		store := &directoryCommandStoreStub{batchUpsertErr: errors.New("should not be called")}
		service := NewDirectoryCommandService(store, &directoryTargetLookupStub{err: gorm.ErrRecordNotFound})

		affected, err := service.BatchUpsert(context.Background(), 1, []DirectoryUpsertItem{{URL: "https://example.com/a"}})
		if !errors.Is(err, ErrTargetNotFound) {
			t.Fatalf("expected ErrTargetNotFound, got affected=%d err=%v", affected, err)
		}
		if store.batchUpsertCalls != 0 {
			t.Fatalf("expected store batch upsert to be skipped, got %d calls", store.batchUpsertCalls)
		}
	})

	t.Run("unexpected lookup error passes through", func(t *testing.T) {
		wantErr := errors.New("lookup failed")
		store := &directoryCommandStoreStub{batchUpsertErr: errors.New("should not be called")}
		service := NewDirectoryCommandService(store, &directoryTargetLookupStub{err: wantErr})

		affected, err := service.BatchUpsert(context.Background(), 1, []DirectoryUpsertItem{{URL: "https://example.com/a"}})
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected lookup error %v, got affected=%d err=%v", wantErr, affected, err)
		}
		if store.batchUpsertCalls != 0 {
			t.Fatalf("expected store batch upsert to be skipped, got %d calls", store.batchUpsertCalls)
		}
	})
}

func TestDirectoryCommandServiceStoreErrorsPropagate(t *testing.T) {
	lookup := &directoryTargetLookupStub{
		targets: map[int]*assetdomain.TargetRef{
			1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
		},
	}

	tests := []struct {
		name    string
		execute func(*DirectoryCommandService) error
	}{
		{
			name: "batch create",
			execute: func(service *DirectoryCommandService) error {
				_, err := service.BatchCreate(context.Background(), 1, []string{"https://example.com/a"})
				return err
			},
		},
		{
			name: "batch upsert",
			execute: func(service *DirectoryCommandService) error {
				_, err := service.BatchUpsert(context.Background(), 1, []DirectoryUpsertItem{{URL: "https://example.com/a"}})
				return err
			},
		},
		{
			name: "batch delete",
			execute: func(service *DirectoryCommandService) error {
				_, err := service.BatchDelete(context.Background(), []int{1})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := errors.New(tt.name + " store failed")
			store := &directoryCommandStoreStub{
				batchCreateErr: wantErr,
				batchDeleteErr: wantErr,
				batchUpsertErr: wantErr,
			}
			service := NewDirectoryCommandService(store, lookup)

			if err := tt.execute(service); !errors.Is(err, wantErr) {
				t.Fatalf("expected store error %v, got %v", wantErr, err)
			}
		})
	}
}
