package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type screenshotCommandStoreStub struct {
	deletedIDs []int
	upsertIn   []assetdomain.Screenshot
	deleteErr  error
	upsertErr  error
}

func (stub *screenshotCommandStoreStub) BulkDelete(ids []int) (int64, error) {
	if stub.deleteErr != nil {
		return 0, stub.deleteErr
	}
	stub.deletedIDs = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

func (stub *screenshotCommandStoreStub) BulkUpsert(items []assetdomain.Screenshot) (int64, error) {
	if stub.upsertErr != nil {
		return 0, stub.upsertErr
	}
	stub.upsertIn = append([]assetdomain.Screenshot(nil), items...)
	return int64(len(items)), nil
}

func TestScreenshotCommandServiceBulkDeleteAndUpsert(t *testing.T) {
	store := &screenshotCommandStoreStub{}
	lookup := &screenshotTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	service := NewScreenshotCommandService(store, lookup)

	deleted, err := service.BulkDelete(context.Background(), []int{1, 2})
	if err != nil {
		t.Fatalf("bulk delete failed: %v", err)
	}
	if deleted != 2 || len(store.deletedIDs) != 2 {
		t.Fatalf("unexpected delete result deleted=%d ids=%v", deleted, store.deletedIDs)
	}

	status := int16(200)
	affected, err := service.BulkUpsert(context.Background(), 1, &BulkUpsertScreenshotRequest{Screenshots: []ScreenshotItem{
		{URL: "https://example.com/a", StatusCode: &status},
		{URL: "https://foo.com/a", StatusCode: &status},
	}})
	if err != nil {
		t.Fatalf("bulk upsert failed: %v", err)
	}
	if affected != 1 || len(store.upsertIn) != 1 {
		t.Fatalf("unexpected upsert result affected=%d size=%d", affected, len(store.upsertIn))
	}
}

func TestScreenshotCommandServiceTargetNotFound(t *testing.T) {
	service := NewScreenshotCommandService(&screenshotCommandStoreStub{}, &screenshotTargetLookupStub{err: gorm.ErrRecordNotFound})

	_, err := service.BulkUpsert(context.Background(), 9, &BulkUpsertScreenshotRequest{Screenshots: []ScreenshotItem{{URL: "https://a.com"}}})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}
