package directory

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

type directoryStoreStub struct {
	items           []assetdomain.Directory
	total           int64
	count           int64
	listErr         error
	streamErr       error
	countErr        error
	scanErr         error
	batchCreateErr  error
	batchDeleteErr  error
	batchUpsertErr  error
	deletedIDs      []int
	lastPageSize    int
	lastFilter      string
	lastOrderBy     string
	lastOptionField string
}

func (stub *directoryStoreStub) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Directory, int64, error) {
	_ = targetID
	_ = page
	stub.lastPageSize = pageSize
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]assetdomain.Directory(nil), stub.items...), stub.total, nil
}

func (stub *directoryStoreStub) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	_ = targetID
	stub.lastOptionField = field
	return []assetdomain.FilterOption{{Value: "200", Label: "200", Count: 1}}, nil
}

func (stub *directoryStoreStub) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Directory) error) error {
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

func (stub *directoryStoreStub) CountByTargetID(targetID int) (int64, error) {
	_ = targetID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func (stub *directoryStoreStub) BatchCreate(items []assetdomain.Directory) (int, error) {
	if stub.batchCreateErr != nil {
		return 0, stub.batchCreateErr
	}
	return len(items), nil
}

func (stub *directoryStoreStub) BatchDelete(ids []int) (int64, error) {
	if stub.batchDeleteErr != nil {
		return 0, stub.batchDeleteErr
	}
	stub.deletedIDs = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

func (stub *directoryStoreStub) BatchUpsert(items []assetdomain.Directory) (int64, error) {
	if stub.batchUpsertErr != nil {
		return 0, stub.batchUpsertErr
	}
	return int64(len(items)), nil
}

func (stub *directoryStoreStub) BatchUpsertContext(_ context.Context, items []assetdomain.Directory) (int64, error) {
	return stub.BatchUpsert(items)
}

type directoryLookupStub struct {
	target *assetdomain.TargetRef
	err    error
}

func (stub *directoryLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	if stub.target == nil || stub.target.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	copyTarget := *stub.target
	return &copyTarget, nil
}

func (stub *directoryLookupStub) GetActiveByIDContext(_ context.Context, id int) (*assetdomain.TargetRef, error) {
	return stub.GetActiveByID(id)
}

func TestDirectoryHandlerCoverage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	status := 200
	length := int64(256)
	duration := int64(42)
	store := &directoryStoreStub{
		items: []assetdomain.Directory{{ID: 1, TargetID: 1, URL: "https://example.com/a", Status: &status, ContentLength: &length, ContentType: "text/html", Duration: &duration, CreatedAt: now}},
		total: 1,
		count: 1,
	}
	lookup := &directoryLookupStub{target: &assetdomain.TargetRef{ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}
	handler := NewDirectoryHandler(service.NewDirectoryFacade(service.NewDirectoryQueryService(store, lookup), service.NewDirectoryCommandService(store, lookup)))

	if handler == nil || handler.svc == nil {
		t.Fatal("expected directory handler")
	}

	t.Run("list invalid target", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.List, http.MethodGet, "/v1/targets/bad/directories", gin.Params{{Key: "target", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list invalid query", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.List, http.MethodGet, "/v1/targets/1/directories?pageSize=abc", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list target not found", func(t *testing.T) {
		lookup.err = gorm.ErrRecordNotFound
		recorder := performDirectoryRequest(t, handler.List, http.MethodGet, "/v1/targets/1/directories", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
		lookup.err = nil
	})

	t.Run("list success", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.List, http.MethodGet, `/v1/targets/1/directories?pageSize=20&filter=url%3D%22admin%22&orderBy=contentLength%20desc`, gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "https://example.com/a") || !strings.Contains(recorder.Body.String(), "totalSize") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
		if store.lastPageSize != 20 || store.lastFilter != `url="admin"` || store.lastOrderBy != "contentLength desc" {
			t.Fatalf("unexpected list args size=%d filter=%q orderBy=%q", store.lastPageSize, store.lastFilter, store.lastOrderBy)
		}
	})

	t.Run("list rejects legacy page before store access", func(t *testing.T) {
		store.lastOrderBy = ""
		recorder := performDirectoryRequest(t, handler.List, http.MethodGet, "/v1/targets/1/directories?page=2", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
		if store.lastOrderBy != "" {
			t.Fatalf("legacy page must be rejected before store access, got orderBy=%q", store.lastOrderBy)
		}
	})

	t.Run("list rejects unsupported orderBy", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.List, http.MethodGet, "/v1/targets/1/directories?orderBy=url%20desc", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("filter options success", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.FilterOptions, http.MethodGet, "/v1/targets/1/directories/filterOptions?field=status", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		body := recorder.Body.String()
		if !strings.Contains(body, `"results"`) || !strings.Contains(body, `"value":"200"`) || !strings.Contains(body, `"count":1`) {
			t.Fatalf("unexpected options response: %s", body)
		}
		if store.lastOptionField != "status" {
			t.Fatalf("expected status option field, got %q", store.lastOptionField)
		}
	})

	t.Run("filter options rejects unsupported field", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.FilterOptions, http.MethodGet, "/v1/targets/1/directories/filterOptions?field=url", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("list internal error", func(t *testing.T) {
		store.listErr = fmt.Errorf("list failed")
		recorder := performDirectoryRequest(t, handler.List, http.MethodGet, "/v1/targets/1/directories", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.listErr = nil
	})

	t.Run("batch create bind failure", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/directories:batchCreate", gin.Params{{Key: "target", Value: "1"}}, "{}", "text/plain")
		if recorder.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("expected 415, got %d", recorder.Code)
		}
	})

	t.Run("batch create invalid target id", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/bad/directories:batchCreate", gin.Params{{Key: "target", Value: "bad"}}, `{"urls":["https://example.com/a"]}`, "application/json")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("batch create invalid url is a client error", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/directories:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"urls":["https:%zz"]}`, "application/json")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("batch create success", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/directories:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"urls":["https://example.com/a"]}`, "application/json")
		if recorder.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", recorder.Code)
		}
	})

	t.Run("batch create target not found", func(t *testing.T) {
		lookup.err = gorm.ErrRecordNotFound
		recorder := performDirectoryRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/directories:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"urls":["https://example.com/a"]}`, "application/json")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
		lookup.err = nil
	})

	t.Run("batch create internal error", func(t *testing.T) {
		store.batchCreateErr = fmt.Errorf("create failed")
		recorder := performDirectoryRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/directories:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"urls":["https://example.com/a"]}`, "application/json")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.batchCreateErr = nil
	})

	t.Run("batch delete bind failure", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.BatchDelete, http.MethodPost, "/v1/directories:batchDelete", nil, "{}", "text/plain")
		if recorder.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("expected 415, got %d", recorder.Code)
		}
	})

	t.Run("batch delete success", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.BatchDelete, http.MethodPost, "/v1/directories:batchDelete", nil, `{"names":["targets/1/directories/1","targets/1/directories/2"]}`, "application/json")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "\"deletedCount\":2") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("batch delete internal error", func(t *testing.T) {
		store.batchDeleteErr = fmt.Errorf("delete failed")
		recorder := performDirectoryRequest(t, handler.BatchDelete, http.MethodPost, "/v1/directories:batchDelete", nil, `{"names":["targets/1/directories/1","targets/1/directories/2"]}`, "application/json")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.batchDeleteErr = nil
	})

	t.Run("export invalid id", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.Export, http.MethodGet, "/v1/targets/bad/directories/exportFiles/current", gin.Params{{Key: "target", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("export target not found", func(t *testing.T) {
		lookup.err = gorm.ErrRecordNotFound
		recorder := performDirectoryRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/directories/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
		lookup.err = nil
	})

	t.Run("export count error", func(t *testing.T) {
		store.countErr = fmt.Errorf("count failed")
		recorder := performDirectoryRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/directories/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.countErr = nil
	})

	t.Run("export success", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/directories/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "content_length,content_type,duration") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("export stream error", func(t *testing.T) {
		store.streamErr = fmt.Errorf("stream failed")
		recorder := performDirectoryRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/directories/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "target_id,url,status") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
		store.streamErr = nil
	})

	t.Run("export scan error", func(t *testing.T) {
		store.scanErr = fmt.Errorf("scan failed")
		recorder := performDirectoryRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/directories/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "target_id,url,status") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
		store.scanErr = nil
	})
}

func performDirectoryRequest(t *testing.T, handlerFunc gin.HandlerFunc, method, target string, params gin.Params, body, contentType string) *httptest.ResponseRecorder {
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
