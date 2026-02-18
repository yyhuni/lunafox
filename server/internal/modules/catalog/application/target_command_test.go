package application

import (
	"context"
	"errors"
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"gorm.io/gorm"
)

type targetCommandStoreStub struct {
	targetByID       map[int]*catalogdomain.Target
	nameExists       map[string]bool
	createdTarget    *catalogdomain.Target
	updatedTarget    *catalogdomain.Target
	createdBatch     []catalogdomain.Target
	findByNamesItems []catalogdomain.Target
	findByIDErr      error
	createErr        error
	updateErr        error
	softDeleteErr    error
	bulkDeleteErr    error
	bulkCreateErr    error
	findByNamesErr   error
	deletedID        int
	bulkDeletedIDs   []int
}

func (stub *targetCommandStoreStub) GetActiveByID(id int) (*catalogdomain.Target, error) {
	if stub.findByIDErr != nil {
		return nil, stub.findByIDErr
	}
	target, ok := stub.targetByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyTarget := *target
	return &copyTarget, nil
}

func (stub *targetCommandStoreStub) ExistsByName(name string, excludeID ...int) (bool, error) {
	_ = excludeID
	return stub.nameExists[name], nil
}

func (stub *targetCommandStoreStub) Create(target *catalogdomain.Target) error {
	if stub.createErr != nil {
		return stub.createErr
	}
	copyTarget := *target
	stub.createdTarget = &copyTarget
	return nil
}

func (stub *targetCommandStoreStub) Update(target *catalogdomain.Target) error {
	if stub.updateErr != nil {
		return stub.updateErr
	}
	copyTarget := *target
	stub.updatedTarget = &copyTarget
	return nil
}

func (stub *targetCommandStoreStub) SoftDelete(id int) error {
	if stub.softDeleteErr != nil {
		return stub.softDeleteErr
	}
	stub.deletedID = id
	return nil
}

func (stub *targetCommandStoreStub) BulkSoftDelete(ids []int) (int64, error) {
	if stub.bulkDeleteErr != nil {
		return 0, stub.bulkDeleteErr
	}
	stub.bulkDeletedIDs = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

func (stub *targetCommandStoreStub) BulkCreateIgnoreConflicts(targets []catalogdomain.Target) (int, error) {
	if stub.bulkCreateErr != nil {
		return 0, stub.bulkCreateErr
	}
	stub.createdBatch = append([]catalogdomain.Target(nil), targets...)
	return len(targets), nil
}

func (stub *targetCommandStoreStub) FindByNames(names []string) ([]catalogdomain.Target, error) {
	_ = names
	if stub.findByNamesErr != nil {
		return nil, stub.findByNamesErr
	}
	return append([]catalogdomain.Target(nil), stub.findByNamesItems...), nil
}

type organizationStoreStub struct {
	existsResult bool
	existsErr    error
	bindErr      error
	boundOrgID   int
	boundIDs     []int
}

func (stub *organizationStoreStub) ExistsByID(id int) (bool, error) {
	_ = id
	if stub.existsErr != nil {
		return false, stub.existsErr
	}
	return stub.existsResult, nil
}

func (stub *organizationStoreStub) BulkAddTargets(organizationID int, targetIDs []int) error {
	if stub.bindErr != nil {
		return stub.bindErr
	}
	stub.boundOrgID = organizationID
	stub.boundIDs = append([]int(nil), targetIDs...)
	return nil
}

func TestTargetCommandServiceCreateAndUpdate(t *testing.T) {
	store := &targetCommandStoreStub{
		targetByID: map[int]*catalogdomain.Target{1: {ID: 1, Name: "example.com", Type: "domain"}},
		nameExists: map[string]bool{},
	}
	service := NewTargetCommandService(store, nil)

	target, err := service.CreateTarget(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("create target failed: %v", err)
	}
	if target.Type != "domain" {
		t.Fatalf("expected domain type, got %s", target.Type)
	}

	updated, err := service.UpdateTarget(context.Background(), 1, "1.1.1.1")
	if err != nil {
		t.Fatalf("update target failed: %v", err)
	}
	if updated.Type != "ip" {
		t.Fatalf("expected ip type, got %s", updated.Type)
	}
}

func TestTargetCommandServiceDeleteAndBulkDelete(t *testing.T) {
	store := &targetCommandStoreStub{
		targetByID: map[int]*catalogdomain.Target{7: {ID: 7, Name: "a.com", Type: "domain"}},
	}
	service := NewTargetCommandService(store, nil)

	if err := service.DeleteTarget(context.Background(), 7); err != nil {
		t.Fatalf("delete target failed: %v", err)
	}
	if store.deletedID != 7 {
		t.Fatalf("expected deleted id 7, got %d", store.deletedID)
	}

	count, err := service.BulkDeleteTargets(context.Background(), []int{1, 2, 3})
	if err != nil {
		t.Fatalf("bulk delete failed: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected deleted count 3, got %d", count)
	}
}

func TestTargetCommandServiceBatchCreateTargets(t *testing.T) {
	t.Run("仅无效目标", func(t *testing.T) {
		service := NewTargetCommandService(&targetCommandStoreStub{}, nil)
		result := service.BatchCreateTargets(context.Background(), []string{"***"}, nil)
		if result.CreatedCount != 0 || result.FailedCount != 1 {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("组织不存在", func(t *testing.T) {
		orgID := 9
		orgStore := &organizationStoreStub{existsResult: false}
		service := NewTargetCommandService(&targetCommandStoreStub{}, orgStore)

		result := service.BatchCreateTargets(context.Background(), []string{"example.com"}, &orgID)
		if result.Message != "organization not found" {
			t.Fatalf("unexpected message: %s", result.Message)
		}
	})

	t.Run("创建并关联组织", func(t *testing.T) {
		orgID := 1
		store := &targetCommandStoreStub{
			findByNamesItems: []catalogdomain.Target{{ID: 11, Name: "example.com", Type: "domain"}},
		}
		orgStore := &organizationStoreStub{existsResult: true}
		service := NewTargetCommandService(store, orgStore)

		result := service.BatchCreateTargets(context.Background(), []string{"example.com", "example.com"}, &orgID)
		if result.CreatedCount != 1 {
			t.Fatalf("expected created 1, got %d", result.CreatedCount)
		}
		if orgStore.boundOrgID != 1 || len(orgStore.boundIDs) != 1 || orgStore.boundIDs[0] != 11 {
			t.Fatalf("unexpected organization binding: org=%d ids=%v", orgStore.boundOrgID, orgStore.boundIDs)
		}
	})

	t.Run("组织验证报错", func(t *testing.T) {
		orgID := 2
		orgStore := &organizationStoreStub{existsErr: errors.New("db error")}
		service := NewTargetCommandService(&targetCommandStoreStub{}, orgStore)

		result := service.BatchCreateTargets(context.Background(), []string{"example.com"}, &orgID)
		if result.CreatedCount != 0 || result.FailedCount != 1 {
			t.Fatalf("unexpected result: %+v", result)
		}
	})
}
