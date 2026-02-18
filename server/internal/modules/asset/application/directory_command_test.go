package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type directoryCommandStoreStub struct {
	bulkCreateIn   []assetdomain.Directory
	bulkUpsertIn   []assetdomain.Directory
	bulkDeletedIDs []int
	bulkCreateErr  error
	bulkDeleteErr  error
	bulkUpsertErr  error
}

func (stub *directoryCommandStoreStub) BulkCreate(items []assetdomain.Directory) (int, error) {
	if stub.bulkCreateErr != nil {
		return 0, stub.bulkCreateErr
	}
	stub.bulkCreateIn = append([]assetdomain.Directory(nil), items...)
	return len(items), nil
}

func (stub *directoryCommandStoreStub) BulkDelete(ids []int) (int64, error) {
	if stub.bulkDeleteErr != nil {
		return 0, stub.bulkDeleteErr
	}
	stub.bulkDeletedIDs = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

func (stub *directoryCommandStoreStub) BulkUpsert(items []assetdomain.Directory) (int64, error) {
	if stub.bulkUpsertErr != nil {
		return 0, stub.bulkUpsertErr
	}
	stub.bulkUpsertIn = append([]assetdomain.Directory(nil), items...)
	return int64(len(items)), nil
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

func TestDirectoryCommandServiceBulkCreateAndUpsert(t *testing.T) {
	store := &directoryCommandStoreStub{}
	lookup := &directoryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewDirectoryCommandService(store, lookup)

	created, err := service.BulkCreate(context.Background(), 1, []string{"https://example.com/a", "https://foo.com"})
	if err != nil {
		t.Fatalf("bulk create failed: %v", err)
	}
	if created != 1 || len(store.bulkCreateIn) != 1 {
		t.Fatalf("unexpected create result created=%d size=%d", created, len(store.bulkCreateIn))
	}

	affected, err := service.BulkUpsert(context.Background(), 1, []DirectoryUpsertItem{{URL: "https://example.com/x"}, {URL: "https://foo.com/x"}})
	if err != nil {
		t.Fatalf("bulk upsert failed: %v", err)
	}
	if affected != 1 || len(store.bulkUpsertIn) != 1 {
		t.Fatalf("unexpected upsert result affected=%d size=%d", affected, len(store.bulkUpsertIn))
	}
}

func TestDirectoryCommandServiceErrorsAndBulkDelete(t *testing.T) {
	store := &directoryCommandStoreStub{}
	lookup := &directoryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewDirectoryCommandService(store, lookup)

	lookup.err = gorm.ErrRecordNotFound
	_, err := service.BulkCreate(context.Background(), 9, []string{"https://a.com"})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}

	lookup.err = nil
	deleted, err := service.BulkDelete(context.Background(), []int{1, 2, 3})
	if err != nil {
		t.Fatalf("bulk delete failed: %v", err)
	}
	if deleted != 3 || len(store.bulkDeletedIDs) != 3 {
		t.Fatalf("unexpected bulk delete result deleted=%d ids=%v", deleted, store.bulkDeletedIDs)
	}
}
