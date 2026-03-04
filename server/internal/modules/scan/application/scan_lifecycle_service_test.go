package application

import (
	"context"
	"errors"
	"testing"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

type lifecycleScanStoreStub struct {
	updateStatusErr error
}

func (stub *lifecycleScanStoreStub) GetByIDNotDeleted(id int) (*QueryScan, error) {
	return nil, nil
}

func (stub *lifecycleScanStoreStub) FindByIDs(ids []int) ([]QueryScan, error) {
	return nil, nil
}

func (stub *lifecycleScanStoreStub) CreateWithTasks(scan *CreateScan, tasks []CreateScanTask) error {
	return nil
}

func (stub *lifecycleScanStoreStub) BulkSoftDelete(ids []int) (int64, []string, error) {
	return 0, nil, nil
}

func (stub *lifecycleScanStoreStub) UpdateStatus(id int, status string, errorMessage ...string) error {
	return stub.updateStatusErr
}

func TestLifecycleServiceStopActiveForDeleteIgnoresCannotStop(t *testing.T) {
	store := &lifecycleScanStoreStub{updateStatusErr: scandomain.ErrScanCannotStop}
	service := NewLifecycleService(store, nil, nil, nil)

	before := scanDeleteStopIgnoredTotal.Value()
	_, err := service.stopActiveForDelete(context.Background(), &QueryScan{
		ID:     1001,
		Status: string(scandomain.ScanStatusRunning),
	})
	after := scanDeleteStopIgnoredTotal.Value()

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if after != before+1 {
		t.Fatalf("expected metric +1, before=%d after=%d", before, after)
	}
}

func TestLifecycleServiceStopActiveForDeleteIgnoresInvalidStatusChange(t *testing.T) {
	store := &lifecycleScanStoreStub{updateStatusErr: scandomain.ErrInvalidStatusChange}
	service := NewLifecycleService(store, nil, nil, nil)

	before := scanDeleteStopIgnoredTotal.Value()
	_, err := service.stopActiveForDelete(context.Background(), &QueryScan{
		ID:     1002,
		Status: string(scandomain.ScanStatusRunning),
	})
	after := scanDeleteStopIgnoredTotal.Value()

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if after != before+1 {
		t.Fatalf("expected metric +1, before=%d after=%d", before, after)
	}
}

func TestLifecycleServiceStopActiveForDeleteReturnsUnexpectedError(t *testing.T) {
	unexpected := errors.New("boom")
	store := &lifecycleScanStoreStub{updateStatusErr: unexpected}
	service := NewLifecycleService(store, nil, nil, nil)

	before := scanDeleteStopIgnoredTotal.Value()
	_, err := service.stopActiveForDelete(context.Background(), &QueryScan{
		ID:     1003,
		Status: string(scandomain.ScanStatusRunning),
	})
	after := scanDeleteStopIgnoredTotal.Value()

	if !errors.Is(err, unexpected) {
		t.Fatalf("expected unexpected error, got %v", err)
	}
	if after != before {
		t.Fatalf("expected metric unchanged, before=%d after=%d", before, after)
	}
}
