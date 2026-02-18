package application

import (
	"context"
	"errors"
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"gorm.io/gorm"
)

type engineCommandStoreStub struct {
	engineByID      map[int]*catalogdomain.ScanEngine
	nameExists      map[string]bool
	createdEngine   *catalogdomain.ScanEngine
	updatedEngine   *catalogdomain.ScanEngine
	findByIDErr     error
	existsByNameErr error
	createErr       error
	updateErr       error
	deleteErr       error
	deletedID       int
}

func (stub *engineCommandStoreStub) GetByID(id int) (*catalogdomain.ScanEngine, error) {
	if stub.findByIDErr != nil {
		return nil, stub.findByIDErr
	}
	engine, ok := stub.engineByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyEngine := *engine
	return &copyEngine, nil
}

func (stub *engineCommandStoreStub) ExistsByName(name string, excludeID ...int) (bool, error) {
	if stub.existsByNameErr != nil {
		return false, stub.existsByNameErr
	}
	_ = excludeID
	return stub.nameExists[name], nil
}

func (stub *engineCommandStoreStub) Create(engine *catalogdomain.ScanEngine) error {
	if stub.createErr != nil {
		return stub.createErr
	}
	copyEngine := *engine
	stub.createdEngine = &copyEngine
	return nil
}

func (stub *engineCommandStoreStub) Update(engine *catalogdomain.ScanEngine) error {
	if stub.updateErr != nil {
		return stub.updateErr
	}
	copyEngine := *engine
	stub.updatedEngine = &copyEngine
	return nil
}

func (stub *engineCommandStoreStub) Delete(id int) error {
	if stub.deleteErr != nil {
		return stub.deleteErr
	}
	stub.deletedID = id
	return nil
}

func TestEngineCommandServiceCreateEngine(t *testing.T) {
	t.Run("空名称非法", func(t *testing.T) {
		service := NewEngineCommandService(&engineCommandStoreStub{})
		_, err := service.CreateEngine(context.Background(), "   ", "{}")
		if !errors.Is(err, ErrInvalidEngine) {
			t.Fatalf("expected ErrInvalidEngine, got %v", err)
		}
	})

	t.Run("名称重复", func(t *testing.T) {
		store := &engineCommandStoreStub{nameExists: map[string]bool{"nmap": true}}
		service := NewEngineCommandService(store)

		_, err := service.CreateEngine(context.Background(), "nmap", "{}")
		if !errors.Is(err, ErrEngineExists) {
			t.Fatalf("expected ErrEngineExists, got %v", err)
		}
	})

	t.Run("成功创建", func(t *testing.T) {
		store := &engineCommandStoreStub{nameExists: map[string]bool{}}
		service := NewEngineCommandService(store)

		engine, err := service.CreateEngine(context.Background(), "  nmap  ", "{}")
		if err != nil {
			t.Fatalf("create engine failed: %v", err)
		}
		if store.createdEngine == nil {
			t.Fatalf("expected store.Create called")
		}
		if engine.Name != "nmap" {
			t.Fatalf("expected trimmed name, got %q", engine.Name)
		}
	})
}

func TestEngineCommandServiceUpdateAndDelete(t *testing.T) {
	t.Run("更新命中不存在", func(t *testing.T) {
		store := &engineCommandStoreStub{engineByID: map[int]*catalogdomain.ScanEngine{}}
		service := NewEngineCommandService(store)

		_, err := service.UpdateEngine(context.Background(), 9, "naabu", "{}")
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
		}
	})

	t.Run("删除成功", func(t *testing.T) {
		store := &engineCommandStoreStub{engineByID: map[int]*catalogdomain.ScanEngine{3: {ID: 3, Name: "x"}}}
		service := NewEngineCommandService(store)

		if err := service.DeleteEngine(context.Background(), 3); err != nil {
			t.Fatalf("delete failed: %v", err)
		}
		if store.deletedID != 3 {
			t.Fatalf("expected deleted id 3, got %d", store.deletedID)
		}
	})
}
