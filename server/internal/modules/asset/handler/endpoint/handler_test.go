package endpoint

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type endpointStoreStub struct {
	items           []assetdomain.Endpoint
	total           int64
	count           int64
	endpointByID    map[int]*assetdomain.Endpoint
	streamErr       error
	countErr        error
	listErr         error
	scanErr         error
	getErr          error
	batchCreateErr  error
	deleteErr       error
	batchDeleteErr  error
	batchUpsertErr  error
	deletedID       int
	batchDeletedIDs []int
}

func (stub *endpointStoreStub) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Endpoint, int64, error) {
	_ = targetID
	_ = page
	_ = pageSize
	_ = filter
	_ = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]assetdomain.Endpoint(nil), stub.items...), stub.total, nil
}

func (stub *endpointStoreStub) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	_ = targetID
	_ = field
	return []assetdomain.FilterOption{{Value: "nginx", Label: "nginx", Count: 1}}, nil
}

func (stub *endpointStoreStub) GetByID(id int) (*assetdomain.Endpoint, error) {
	if stub.getErr != nil {
		return nil, stub.getErr
	}
	item, ok := stub.endpointByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyItem := *item
	return &copyItem, nil
}

func (stub *endpointStoreStub) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Endpoint) error) error {
	_ = ctx
	_ = targetID
	if stub.streamErr != nil {
		return stub.streamErr
	}
	if stub.scanErr != nil {
		return stub.scanErr
	}
	for _, item := range stub.items {
		if err := visit(item); err != nil {
			return err
		}
	}
	return nil
}

func (stub *endpointStoreStub) CountByTargetID(targetID int) (int64, error) {
	_ = targetID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func (stub *endpointStoreStub) BatchCreate(endpoints []assetdomain.Endpoint) (int, error) {
	if stub.batchCreateErr != nil {
		return 0, stub.batchCreateErr
	}
	return len(endpoints), nil
}

func (stub *endpointStoreStub) Delete(id int) error {
	if stub.deleteErr != nil {
		return stub.deleteErr
	}
	stub.deletedID = id
	return nil
}

func (stub *endpointStoreStub) BatchDelete(ids []int) (int64, error) {
	if stub.batchDeleteErr != nil {
		return 0, stub.batchDeleteErr
	}
	stub.batchDeletedIDs = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

func (stub *endpointStoreStub) BatchUpsert(endpoints []assetdomain.Endpoint) (int64, error) {
	if stub.batchUpsertErr != nil {
		return 0, stub.batchUpsertErr
	}
	return int64(len(endpoints)), nil
}

type endpointLookupStub struct {
	target *assetdomain.TargetRef
	err    error
}

func (stub *endpointLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	if stub.target == nil || stub.target.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	copyTarget := *stub.target
	return &copyTarget, nil
}

func TestEndpointHandlerCoverage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	status := 200
	contentLength := 512
	vhost := true
	store := &endpointStoreStub{
		items: []assetdomain.Endpoint{{
			ID:              1,
			TargetID:        1,
			URL:             "https://example.com/api",
			Host:            "example.com",
			Title:           "api",
			StatusCode:      &status,
			ContentLength:   &contentLength,
			Tech:            []string{"go", "gin"},
			Vhost:           &vhost,
			ResponseHeaders: "server: nginx",
			CreatedAt:       now,
		}},
		total: 1,
		count: 1,
		endpointByID: map[int]*assetdomain.Endpoint{
			2: {ID: 2, TargetID: 1, URL: "https://example.com/api/2", Host: "example.com"},
		},
	}
	lookup := &endpointLookupStub{target: &assetdomain.TargetRef{ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}
	handler := NewEndpointHandler(service.NewEndpointFacade(service.NewEndpointQueryService(store, lookup), service.NewEndpointCommandService(store, lookup)))

	if handler == nil || handler.svc == nil {
		t.Fatal("expected endpoint handler to be initialized")
	}

	t.Run("list invalid target", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.List, http.MethodGet, "/v1/targets/bad/endpoints", gin.Params{{Key: "target", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list invalid query", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.List, http.MethodGet, "/v1/targets/1/endpoints?pageSize=abc", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list target not found", func(t *testing.T) {
		lookup.err = gorm.ErrRecordNotFound
		recorder := performEndpointRequest(t, handler.List, http.MethodGet, "/v1/targets/1/endpoints", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
		lookup.err = nil
	})

	t.Run("list success", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.List, http.MethodGet, "/v1/targets/1/endpoints", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		if !strings.Contains(recorder.Body.String(), "https://example.com/api") {
			t.Fatalf("unexpected body: %s", recorder.Body.String())
		}
	})

	t.Run("list internal error", func(t *testing.T) {
		store.listErr = fmt.Errorf("list failed")
		recorder := performEndpointRequest(t, handler.List, http.MethodGet, "/v1/targets/1/endpoints", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.listErr = nil
	})

	t.Run("get invalid id", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.GetByID, http.MethodGet, "/v1/endpoints/bad", gin.Params{{Key: "endpoint", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.GetByID, http.MethodGet, "/v1/endpoints/404", gin.Params{{Key: "endpoint", Value: "404"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
	})

	t.Run("get success", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.GetByID, http.MethodGet, "/v1/endpoints/2", gin.Params{{Key: "endpoint", Value: "2"}}, "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		if !strings.Contains(recorder.Body.String(), "\"id\":2") {
			t.Fatalf("unexpected body: %s", recorder.Body.String())
		}
	})

	t.Run("get internal error", func(t *testing.T) {
		store.getErr = fmt.Errorf("get failed")
		recorder := performEndpointRequest(t, handler.GetByID, http.MethodGet, "/v1/endpoints/2", gin.Params{{Key: "endpoint", Value: "2"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.getErr = nil
	})

	t.Run("batch create bind failure", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/endpoints:batchCreate", gin.Params{{Key: "target", Value: "1"}}, "{}", "text/plain")
		if recorder.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("expected 415, got %d", recorder.Code)
		}
	})

	t.Run("batch create invalid target id", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/bad/endpoints:batchCreate", gin.Params{{Key: "target", Value: "bad"}}, `{"urls":["https://example.com/api"]}`, "application/json")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("batch create invalid url is a client error", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/endpoints:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"urls":["https:%zz"]}`, "application/json")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("batch create success", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/endpoints:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"urls":["https://example.com/api"]}`, "application/json")
		if recorder.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", recorder.Code)
		}
	})

	t.Run("batch create target not found", func(t *testing.T) {
		lookup.err = gorm.ErrRecordNotFound
		recorder := performEndpointRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/endpoints:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"urls":["https://example.com/api"]}`, "application/json")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
		lookup.err = nil
	})

	t.Run("batch create internal error", func(t *testing.T) {
		store.batchCreateErr = fmt.Errorf("create failed")
		recorder := performEndpointRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/endpoints:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"urls":["https://example.com/api"]}`, "application/json")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.batchCreateErr = nil
	})

	t.Run("delete invalid id", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.Delete, http.MethodDelete, "/v1/endpoints/bad", gin.Params{{Key: "endpoint", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("delete not found", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.Delete, http.MethodDelete, "/v1/endpoints/404", gin.Params{{Key: "endpoint", Value: "404"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
	})

	t.Run("delete success", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.Delete, http.MethodDelete, "/v1/endpoints/2", gin.Params{{Key: "endpoint", Value: "2"}}, "", "")
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", recorder.Code)
		}
		if store.deletedID != 2 {
			t.Fatalf("expected deleted id 2, got %d", store.deletedID)
		}
	})

	t.Run("delete internal error", func(t *testing.T) {
		store.getErr = fmt.Errorf("get failed")
		recorder := performEndpointRequest(t, handler.Delete, http.MethodDelete, "/v1/endpoints/2", gin.Params{{Key: "endpoint", Value: "2"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.getErr = nil
	})

	t.Run("batch delete bind failure", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.BatchDelete, http.MethodPost, "/v1/endpoints:batchDelete", nil, "{}", "text/plain")
		if recorder.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("expected 415, got %d", recorder.Code)
		}
	})

	t.Run("batch delete success", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.BatchDelete, http.MethodPost, "/v1/endpoints:batchDelete", nil, `{"names":["targets/1/endpoints/1","targets/1/endpoints/2"]}`, "application/json")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
	})

	t.Run("batch delete internal error", func(t *testing.T) {
		store.batchDeleteErr = fmt.Errorf("delete failed")
		recorder := performEndpointRequest(t, handler.BatchDelete, http.MethodPost, "/v1/endpoints:batchDelete", nil, `{"names":["targets/1/endpoints/1","targets/1/endpoints/2"]}`, "application/json")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.batchDeleteErr = nil
	})

	t.Run("export invalid id", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.Export, http.MethodGet, "/v1/targets/bad/endpoints/exportFiles/current", gin.Params{{Key: "target", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("export target not found", func(t *testing.T) {
		lookup.err = gorm.ErrRecordNotFound
		recorder := performEndpointRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/endpoints/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
		lookup.err = nil
	})

	t.Run("export success", func(t *testing.T) {
		recorder := performEndpointRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/endpoints/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		body := recorder.Body.String()
		if !strings.Contains(body, "target_id,url,host") || !strings.Contains(body, "go|gin") {
			t.Fatalf("unexpected csv body: %s", body)
		}
	})

	t.Run("export stream error", func(t *testing.T) {
		store.streamErr = fmt.Errorf("stream failed")
		recorder := performEndpointRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/endpoints/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "target_id,url,host") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
		store.streamErr = nil
	})

	t.Run("export count error", func(t *testing.T) {
		store.countErr = fmt.Errorf("count failed")
		recorder := performEndpointRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/endpoints/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.countErr = nil
	})

	t.Run("export scan error", func(t *testing.T) {
		store.scanErr = fmt.Errorf("scan failed")
		recorder := performEndpointRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/endpoints/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "target_id,url,host") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
		store.scanErr = nil
	})
}

func performEndpointRequest(t *testing.T, handlerFunc gin.HandlerFunc, method, target string, params gin.Params, body, contentType string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	ctx.Request = req
	ctx.Params = params
	handlerFunc(ctx)
	ctx.Writer.WriteHeaderNow()
	return recorder
}
