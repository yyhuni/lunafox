package application

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

type workerTargetNameStoreStub struct {
	scan *QueryScan
	err  error
}

func (stub *workerTargetNameStoreStub) FindAll(page, pageSize int, targetID int, status, search string) ([]QueryScan, int64, error) {
	return nil, 0, nil
}

func (stub *workerTargetNameStoreStub) GetQueryByID(id int) (*QueryScan, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return stub.scan, nil
}

func (stub *workerTargetNameStoreStub) GetByIDNotDeleted(id int) (*QueryScan, error) {
	return nil, nil
}

func (stub *workerTargetNameStoreStub) FindByIDs(ids []int) ([]QueryScan, error) {
	return nil, nil
}

func (stub *workerTargetNameStoreStub) CreateWithInputTargetsAndTasks(scan *CreateScan, inputs []CreateScanInputTarget, tasks []CreateScanTask) error {
	return nil
}

func (stub *workerTargetNameStoreStub) BulkSoftDelete(ids []int) (int64, []string, error) {
	return 0, nil, nil
}

func (stub *workerTargetNameStoreStub) GetStatistics() (*QueryStatistics, error) {
	return &QueryStatistics{}, nil
}

func (stub *workerTargetNameStoreStub) UpdateStatus(id int, status string, errorMessage ...string) error {
	return nil
}

func TestScanFacadeGetTargetName(t *testing.T) {
	t.Run("scan not found", func(t *testing.T) {
		facade := NewScanFacade(&workerTargetNameStoreStub{err: gorm.ErrRecordNotFound}, &workerTargetNameStoreStub{err: gorm.ErrRecordNotFound}, nil, nil, nil, nil)
		_, err := facade.GetTargetName(1)
		if !errors.Is(err, ErrScanNotFound) {
			t.Fatalf("expected ErrScanNotFound, got %v", err)
		}
	})

	t.Run("target not found", func(t *testing.T) {
		facade := NewScanFacade(&workerTargetNameStoreStub{scan: &QueryScan{ID: 1}}, &workerTargetNameStoreStub{scan: &QueryScan{ID: 1}}, nil, nil, nil, nil)
		_, err := facade.GetTargetName(1)
		if !errors.Is(err, ErrScanTargetNotFound) {
			t.Fatalf("expected ErrScanTargetNotFound, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		facade := NewScanFacade(&workerTargetNameStoreStub{scan: &QueryScan{ID: 1, Target: &QueryTargetRef{Name: "example.com", Type: "domain"}}}, &workerTargetNameStoreStub{scan: &QueryScan{ID: 1, Target: &QueryTargetRef{Name: "example.com", Type: "domain"}}}, nil, nil, nil, nil)
		target, err := facade.GetTargetName(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if target.Name != "example.com" || target.Type != "domain" {
			t.Fatalf("unexpected target %+v", target)
		}
	})
}
