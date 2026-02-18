package application

import (
	"context"
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type engineQueryStoreStub struct {
	engines  []catalogdomain.ScanEngine
	total    int64
	engine   *catalogdomain.ScanEngine
	listErr  error
	getErr   error
	listPage int
	listSize int
}

func (stub *engineQueryStoreStub) GetByID(id int) (*catalogdomain.ScanEngine, error) {
	_ = id
	if stub.getErr != nil {
		return nil, stub.getErr
	}
	if stub.engine == nil {
		return nil, nil
	}
	copyEngine := *stub.engine
	return &copyEngine, nil
}

func (stub *engineQueryStoreStub) FindAll(page, pageSize int) ([]catalogdomain.ScanEngine, int64, error) {
	stub.listPage = page
	stub.listSize = pageSize
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	result := make([]catalogdomain.ScanEngine, len(stub.engines))
	copy(result, stub.engines)
	return result, stub.total, nil
}

func TestEngineQueryServiceListEngines(t *testing.T) {
	store := &engineQueryStoreStub{
		engines: []catalogdomain.ScanEngine{{ID: 1, Name: "nmap"}},
		total:   1,
	}
	service := NewEngineQueryService(store)

	engines, total, err := service.ListEngines(context.Background(), 2, 20)
	if err != nil {
		t.Fatalf("list engines failed: %v", err)
	}
	if store.listPage != 2 || store.listSize != 20 {
		t.Fatalf("unexpected paging args: page=%d size=%d", store.listPage, store.listSize)
	}
	if len(engines) != 1 || total != 1 {
		t.Fatalf("unexpected list result: len=%d total=%d", len(engines), total)
	}
}

func TestEngineQueryServiceGetEngineByID(t *testing.T) {
	store := &engineQueryStoreStub{engine: &catalogdomain.ScanEngine{ID: 7, Name: "naabu"}}
	service := NewEngineQueryService(store)

	engine, err := service.GetEngineByID(context.Background(), 7)
	if err != nil {
		t.Fatalf("get engine failed: %v", err)
	}
	if engine == nil || engine.ID != 7 || engine.Name != "naabu" {
		t.Fatalf("unexpected engine: %+v", engine)
	}
}
