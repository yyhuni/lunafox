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
	createdTargets   []catalogdomain.Target
	updatedTarget    *catalogdomain.Target
	createdBatch     []catalogdomain.Target
	findByNamesItems []catalogdomain.Target
	findByNamesCalls [][]string
	findByIDErr      error
	createErr        error
	updateErr        error
	softDeleteErr    error
	batchDeleteErr   error
	batchCreateErr   error
	findByNamesErr   error
	deletedID        int
	batchDeletedIDs  []int
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
	if copyTarget.ID == 0 {
		copyTarget.ID = 100 + len(stub.createdTargets)
		target.ID = copyTarget.ID
	}
	stub.createdTarget = &copyTarget
	stub.createdTargets = append(stub.createdTargets, copyTarget)
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

func (stub *targetCommandStoreStub) BatchSoftDelete(ids []int) (int64, error) {
	if stub.batchDeleteErr != nil {
		return 0, stub.batchDeleteErr
	}
	stub.batchDeletedIDs = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

func (stub *targetCommandStoreStub) TombstoneAndEnsureCleanup(_ context.Context, id int) (bool, error) {
	if stub.softDeleteErr != nil {
		return false, stub.softDeleteErr
	}
	stub.deletedID = id
	return true, nil
}

func (stub *targetCommandStoreStub) BatchTombstoneAndEnsureCleanup(_ context.Context, ids []int) (int64, error) {
	if stub.batchDeleteErr != nil {
		return 0, stub.batchDeleteErr
	}
	stub.batchDeletedIDs = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

func (stub *targetCommandStoreStub) BatchCreateIgnoreConflicts(targets []catalogdomain.Target) (int, error) {
	if stub.batchCreateErr != nil {
		return 0, stub.batchCreateErr
	}
	stub.createdBatch = append([]catalogdomain.Target(nil), targets...)
	return len(targets), nil
}

func (stub *targetCommandStoreStub) FindByNames(names []string) ([]catalogdomain.Target, error) {
	stub.findByNamesCalls = append(stub.findByNamesCalls, append([]string(nil), names...))
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

func (stub *organizationStoreStub) BatchAddTargets(organizationID int, targetIDs []int) error {
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

func TestTargetCommandServicePersistsCanonicalIPv4OnlyTargets(t *testing.T) {
	store := &targetCommandStoreStub{
		targetByID: map[int]*catalogdomain.Target{1: {ID: 1, Name: "example.com", Type: catalogdomain.TargetTypeDomain}},
		nameExists: map[string]bool{},
	}
	service := NewTargetCommandService(store, nil)

	created, err := service.CreateTarget(context.Background(), " Example.COM. ")
	if err != nil {
		t.Fatalf("CreateTarget canonical domain: %v", err)
	}
	if created.Name != "example.com" || store.createdTarget == nil || store.createdTarget.Name != "example.com" {
		t.Fatalf("created target was not canonical: result=%#v stored=%#v", created, store.createdTarget)
	}

	updated, err := service.UpdateTarget(context.Background(), 1, "192.0.2.3/24")
	if err != nil {
		t.Fatalf("UpdateTarget canonical CIDR: %v", err)
	}
	if updated.Name != "192.0.2.0/24" || updated.Type != catalogdomain.TargetTypeCIDR || store.updatedTarget == nil || store.updatedTarget.Name != "192.0.2.0/24" {
		t.Fatalf("updated target was not canonical: result=%#v stored=%#v", updated, store.updatedTarget)
	}

	if _, err := service.CreateTarget(context.Background(), "2001:db8::1"); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("CreateTarget IPv6 error = %v, want ErrInvalidTarget", err)
	}
	if _, err := service.UpdateTarget(context.Background(), 1, "2001:db8::/64"); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("UpdateTarget IPv6 error = %v, want ErrInvalidTarget", err)
	}
}

func TestTargetCommandServiceDeleteAndBatchDelete(t *testing.T) {
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

	count, err := service.BatchDeleteTargets(context.Background(), []int{1, 2, 3})
	if err != nil {
		t.Fatalf("batch delete failed: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected deleted count 3, got %d", count)
	}
}

func TestTargetCommandServiceBatchCreateTargets(t *testing.T) {
	t.Run("only invalid targets", func(t *testing.T) {
		service := NewTargetCommandService(&targetCommandStoreStub{}, nil)
		result := service.BatchCreateTargets(context.Background(), []string{"***"}, nil)
		if result.CreatedCount != 0 || result.FailedCount != 1 {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("organization not found", func(t *testing.T) {
		orgID := 9
		orgStore := &organizationStoreStub{existsResult: false}
		service := NewTargetCommandService(&targetCommandStoreStub{}, orgStore)

		result := service.BatchCreateTargets(context.Background(), []string{"example.com"}, &orgID)
		if result.Message != "organization not found" {
			t.Fatalf("unexpected message: %s", result.Message)
		}
	})

	t.Run("create and link organization", func(t *testing.T) {
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

	t.Run("organization validation error", func(t *testing.T) {
		orgID := 2
		orgStore := &organizationStoreStub{existsErr: errors.New("db error")}
		service := NewTargetCommandService(&targetCommandStoreStub{}, orgStore)

		result := service.BatchCreateTargets(context.Background(), []string{"example.com"}, &orgID)
		if result.CreatedCount != 0 || result.FailedCount != 1 {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("canonicalizes and rejects IPv6", func(t *testing.T) {
		store := &targetCommandStoreStub{}
		service := NewTargetCommandService(store, nil)

		result := service.BatchCreateTargets(context.Background(), []string{
			" Example.COM. ",
			"example.com",
			"192.0.2.3/30",
			"2001:db8::1",
		}, nil)
		if result.CreatedCount != 2 || result.FailedCount != 1 {
			t.Fatalf("unexpected batch result: %+v", result)
		}
		if len(store.createdBatch) != 2 || store.createdBatch[0].Name != "example.com" || store.createdBatch[1].Name != "192.0.2.0/30" {
			t.Fatalf("batch targets were not canonical/deduplicated: %#v", store.createdBatch)
		}
	})
}

func TestTargetCommandServiceEnsureTargetsCreatesMissingAndReusesExisting(t *testing.T) {
	store := &targetCommandStoreStub{
		findByNamesItems: []catalogdomain.Target{
			{ID: 7, Name: "test.com", Type: "domain"},
		},
	}
	service := NewTargetCommandService(store, nil)

	result, err := service.EnsureTargets(context.Background(), []string{" Test.COM ", "new.example.com", "new.example.com", "***"})
	if err != nil {
		t.Fatalf("ensure targets returned error: %v", err)
	}

	if len(store.findByNamesCalls) != 1 {
		t.Fatalf("expected one name lookup, got %+v", store.findByNamesCalls)
	}
	if got := store.findByNamesCalls[0]; len(got) != 2 || got[0] != "test.com" || got[1] != "new.example.com" {
		t.Fatalf("unexpected lookup names: %+v", store.findByNamesCalls)
	}
	if len(store.createdTargets) != 1 || store.createdTargets[0].Name != "new.example.com" {
		t.Fatalf("expected only missing target to be created, got %+v", store.createdTargets)
	}
	if len(result.Targets) != 2 {
		t.Fatalf("expected two ensured targets, got %+v", result.Targets)
	}
	if result.Targets[0].Name != "test.com" || result.Targets[1].Name != "new.example.com" {
		t.Fatalf("unexpected ensured target order: %+v", result.Targets)
	}
	if result.TargetStats.Created != 1 || result.TargetStats.Skipped != 1 || result.TargetStats.Failed != 1 {
		t.Fatalf("unexpected ensure target stats: %+v", result.TargetStats)
	}
	if len(result.Errors) != 1 || result.Errors[0].Input != "***" || result.Errors[0].Error == "" {
		t.Fatalf("expected invalid target error, got %+v", result.Errors)
	}
}
