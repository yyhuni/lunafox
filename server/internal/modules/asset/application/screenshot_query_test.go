package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type screenshotQueryStoreStub struct {
	items       []assetdomain.Screenshot
	total       int64
	itemByID    map[int]*assetdomain.Screenshot
	listErr     error
	findByIDErr error
}

func (stub *screenshotQueryStoreStub) FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Screenshot, int64, error) {
	_ = targetID
	_ = page
	_ = pageSize
	_ = filter
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]assetdomain.Screenshot(nil), stub.items...), stub.total, nil
}

func (stub *screenshotQueryStoreStub) GetByID(id int) (*assetdomain.Screenshot, error) {
	if stub.findByIDErr != nil {
		return nil, stub.findByIDErr
	}
	item, ok := stub.itemByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyItem := *item
	return &copyItem, nil
}

type screenshotTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
}

func (stub *screenshotTargetLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
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

func TestScreenshotQueryServiceListAndGetByID(t *testing.T) {
	store := &screenshotQueryStoreStub{
		items:    []assetdomain.Screenshot{{ID: 1}, {ID: 2}},
		total:    2,
		itemByID: map[int]*assetdomain.Screenshot{2: {ID: 2}},
	}
	lookup := &screenshotTargetLookupStub{targets: map[int]*assetdomain.TargetRef{5: {ID: 5}}}
	service := NewScreenshotQueryService(store, lookup)

	items, total, err := service.ListByTargetID(context.Background(), 5, 1, 20, "")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 2 || total != 2 {
		t.Fatalf("unexpected list result len=%d total=%d", len(items), total)
	}

	item, err := service.GetByID(context.Background(), 2)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if item.ID != 2 {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestScreenshotQueryServiceErrors(t *testing.T) {
	service := NewScreenshotQueryService(&screenshotQueryStoreStub{itemByID: map[int]*assetdomain.Screenshot{}}, &screenshotTargetLookupStub{err: gorm.ErrRecordNotFound})

	_, _, err := service.ListByTargetID(context.Background(), 1, 1, 20, "")
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}

	_, err = service.GetByID(context.Background(), 99)
	if !errors.Is(err, ErrScreenshotNotFound) {
		t.Fatalf("expected ErrScreenshotNotFound, got %v", err)
	}
}
