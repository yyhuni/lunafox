package application

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestScanFacadeStopMapsNotFoundAndPreservesCallerContext(t *testing.T) {
	store := &lifecycleScanStopStoreStub{err: gorm.ErrRecordNotFound}
	service := NewLifecycleService(nil, store, nil)
	facade := NewScanFacade(nil, nil, nil, nil, service, nil)
	contextKey := struct{}{}
	ctx := context.WithValue(context.Background(), contextKey, "stop-request")

	_, err := facade.Stop(ctx, 21)
	if !errors.Is(err, ErrScanNotFound) {
		t.Fatalf("facade Stop error = %v, want ErrScanNotFound", err)
	}
	if got := store.ctx.Value(contextKey); got != "stop-request" {
		t.Fatalf("facade did not preserve caller context: %v", got)
	}
}
