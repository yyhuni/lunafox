package application

import (
	"context"
	"errors"
	"testing"
)

type engineCatalogQueryStoreStub struct {
	items   []EngineCatalogItem
	listErr error
	getErr  error
}

func (stub *engineCatalogQueryStoreStub) ListEngines(_ context.Context) ([]EngineCatalogItem, error) {
	if stub.listErr != nil {
		return nil, stub.listErr
	}
	return append([]EngineCatalogItem(nil), stub.items...), nil
}

func (stub *engineCatalogQueryStoreStub) GetEngineByID(_ context.Context, engineID string) (*EngineCatalogItem, error) {
	if stub.getErr != nil {
		return nil, stub.getErr
	}
	for i := range stub.items {
		if stub.items[i].EngineID == engineID {
			item := stub.items[i]
			return &item, nil
		}
	}
	return nil, ErrEngineNotFound
}

func TestEngineCatalogFacadeListsAndGetsInstalledEngines(t *testing.T) {
	store := &engineCatalogQueryStoreStub{items: []EngineCatalogItem{{EngineID: "engine.lunafox.website_discovery"}}}
	facade := NewEngineCatalogFacade(store)
	items, err := facade.ListEngines()
	if err != nil || len(items) != 1 {
		t.Fatalf("ListEngines() = %+v, %v", items, err)
	}
	item, err := facade.GetEngineByID(" engine.lunafox.website_discovery ")
	if err != nil || item.EngineID != "engine.lunafox.website_discovery" {
		t.Fatalf("GetEngineByID() = %+v, %v", item, err)
	}
}

func TestEngineCatalogFacadePropagatesCatalogErrorsAndRejectsEmptyID(t *testing.T) {
	wantErr := errors.New("catalog unavailable")
	facade := NewEngineCatalogFacade(&engineCatalogQueryStoreStub{listErr: wantErr})
	if _, err := facade.ListEngines(); !errors.Is(err, wantErr) {
		t.Fatalf("ListEngines() error = %v", err)
	}
	if _, err := facade.GetEngineByID(" "); !errors.Is(err, ErrEngineNotFound) {
		t.Fatalf("GetEngineByID() error = %v", err)
	}
}
