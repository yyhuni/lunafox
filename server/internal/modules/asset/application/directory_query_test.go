package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type directoryQueryStoreStub struct {
	items        []assetdomain.Directory
	total        int64
	count        int64
	listErr      error
	streamErr    error
	countErr     error
	scannedErr   error
	listTargetID int
	listPage     int
	listPageSize int
	listFilter   string
	streamID     int
	countID      int
}

func (stub *directoryQueryStoreStub) FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Directory, int64, error) {
	stub.listTargetID = targetID
	stub.listPage = page
	stub.listPageSize = pageSize
	stub.listFilter = filter
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]assetdomain.Directory(nil), stub.items...), stub.total, nil
}

func (stub *directoryQueryStoreStub) StreamByTargetID(targetID int) (*sql.Rows, error) {
	stub.streamID = targetID
	if stub.streamErr != nil {
		return nil, stub.streamErr
	}
	return nil, nil
}

func (stub *directoryQueryStoreStub) CountByTargetID(targetID int) (int64, error) {
	stub.countID = targetID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func (stub *directoryQueryStoreStub) ScanRow(rows *sql.Rows) (*assetdomain.Directory, error) {
	if stub.scannedErr != nil {
		return nil, stub.scannedErr
	}
	return &assetdomain.Directory{ID: 11}, nil
}

type directoryQueryTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
}

func (stub *directoryQueryTargetLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
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

func TestDirectoryQueryServiceListAndCount(t *testing.T) {
	store := &directoryQueryStoreStub{
		items: []assetdomain.Directory{{ID: 1}, {ID: 2}},
		total: 2,
		count: 8,
	}
	lookup := &directoryQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{7: {ID: 7}}}
	service := NewDirectoryQueryService(store, lookup)

	items, total, err := service.ListByTarget(context.Background(), 7, 1, 20, "status:200")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 2 || total != 2 {
		t.Fatalf("unexpected list result len=%d total=%d", len(items), total)
	}
	if store.listTargetID != 7 || store.listPage != 1 || store.listPageSize != 20 || store.listFilter != "status:200" {
		t.Fatalf("unexpected list args: %+v", store)
	}

	count, err := service.CountByTarget(context.Background(), 7)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 8 {
		t.Fatalf("expected count 8, got %d", count)
	}
}

func TestDirectoryQueryServiceTargetNotFound(t *testing.T) {
	service := NewDirectoryQueryService(&directoryQueryStoreStub{}, &directoryQueryTargetLookupStub{err: gorm.ErrRecordNotFound})

	_, _, err := service.ListByTarget(context.Background(), 1, 1, 20, "")
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}
