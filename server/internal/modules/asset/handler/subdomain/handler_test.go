package subdomain

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

type subdomainStoreStub struct {
	items          []assetdomain.Subdomain
	total          int64
	count          int64
	listErr        error
	streamErr      error
	countErr       error
	scanErr        error
	batchCreateErr error
	batchDeleteErr error
	lastTargetID   int
	lastPage       int
	lastPageSize   int
	lastFilter     string
	lastOrderBy    string
}

func (stub *subdomainStoreStub) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Subdomain, int64, error) {
	stub.lastTargetID = targetID
	stub.lastPage = page
	stub.lastPageSize = pageSize
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]assetdomain.Subdomain(nil), stub.items...), stub.total, nil
}

func (stub *subdomainStoreStub) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Subdomain) error) error {
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

func (stub *subdomainStoreStub) CountByTargetID(targetID int) (int64, error) {
	_ = targetID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func (stub *subdomainStoreStub) BatchCreate(items []assetdomain.Subdomain) (int, error) {
	if stub.batchCreateErr != nil {
		return 0, stub.batchCreateErr
	}
	return len(items), nil
}

func (stub *subdomainStoreStub) BatchCreateContext(_ context.Context, items []assetdomain.Subdomain) (int, error) {
	return stub.BatchCreate(items)
}

func (stub *subdomainStoreStub) BatchDelete(ids []int) (int64, error) {
	if stub.batchDeleteErr != nil {
		return 0, stub.batchDeleteErr
	}
	return int64(len(ids)), nil
}

type subdomainLookupStub struct {
	target *assetdomain.TargetRef
	err    error
}

func (stub *subdomainLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	if stub.target == nil || stub.target.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	copyTarget := *stub.target
	return &copyTarget, nil
}

func (stub *subdomainLookupStub) GetActiveByIDContext(ctx context.Context, id int) (*assetdomain.TargetRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.GetActiveByID(id)
}

func TestSubdomainHandlerCoverage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	store := &subdomainStoreStub{
		items: []assetdomain.Subdomain{{ID: 1, TargetID: 1, DNSName: "api.example.com", CreatedAt: now}},
		total: 1,
		count: 1,
	}
	lookup := &subdomainLookupStub{target: &assetdomain.TargetRef{ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}
	handler := NewSubdomainHandler(service.NewSubdomainFacade(service.NewSubdomainQueryService(store, lookup), service.NewSubdomainCommandService(store, lookup)))

	if handler == nil || handler.svc == nil {
		t.Fatal("expected subdomain handler")
	}

	t.Run("list invalid target", func(t *testing.T) {
		recorder := performSubdomainRequest(t, handler.List, http.MethodGet, "/v1/targets/bad/subdomains", gin.Params{{Key: "target", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list invalid query", func(t *testing.T) {
		recorder := performSubdomainRequest(t, handler.List, http.MethodGet, "/v1/targets/1/subdomains?pageSize=abc", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list target not found", func(t *testing.T) {
		prevErr := lookup.err
		lookup.err = gorm.ErrRecordNotFound
		t.Cleanup(func() {
			lookup.err = prevErr
		})
		recorder := performSubdomainRequest(t, handler.List, http.MethodGet, "/v1/targets/1/subdomains", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
	})

	t.Run("list success", func(t *testing.T) {
		recorder := performSubdomainRequest(t, handler.List, http.MethodGet, `/v1/targets/1/subdomains?pageSize=1&filter=dnsName%3D%22api%22&orderBy=dnsName%20desc`, gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "api.example.com") || !strings.Contains(recorder.Body.String(), "totalSize") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
		if store.lastTargetID != 1 || store.lastPage != 1 || store.lastPageSize != 1 || store.lastFilter != `dnsName="api"` || store.lastOrderBy != "dnsName desc" {
			t.Fatalf("unexpected list args: target=%d page=%d pageSize=%d filter=%q orderBy=%q", store.lastTargetID, store.lastPage, store.lastPageSize, store.lastFilter, store.lastOrderBy)
		}
	})

	t.Run("list rejects old page parameter before store access", func(t *testing.T) {
		prevTargetID := store.lastTargetID
		recorder := performSubdomainRequest(t, handler.List, http.MethodGet, "/v1/targets/1/subdomains?page=2", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if store.lastTargetID != prevTargetID {
			t.Fatal("legacy page parameter must be rejected before store access")
		}
	})

	t.Run("list rejects unsupported orderBy", func(t *testing.T) {
		recorder := performSubdomainRequest(t, handler.List, http.MethodGet, "/v1/targets/1/subdomains?orderBy=targetCount%20desc", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("list internal error", func(t *testing.T) {
		prevErr := store.listErr
		store.listErr = fmt.Errorf("list failed")
		t.Cleanup(func() {
			store.listErr = prevErr
		})
		recorder := performSubdomainRequest(t, handler.List, http.MethodGet, "/v1/targets/1/subdomains", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
	})

	t.Run("batch create bind failure", func(t *testing.T) {
		recorder := performSubdomainRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/subdomains:batchCreate", gin.Params{{Key: "target", Value: "1"}}, "{}", "text/plain")
		if recorder.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("expected 415, got %d", recorder.Code)
		}
	})

	t.Run("batch create rejects old names payload", func(t *testing.T) {
		recorder := performSubdomainRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/subdomains:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"names":["api.example.com"]}`, "application/json")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("batch create invalid target id", func(t *testing.T) {
		recorder := performSubdomainRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/bad/subdomains:batchCreate", gin.Params{{Key: "target", Value: "bad"}}, `{"dnsNames":["api.example.com"]}`, "application/json")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("batch create invalid target type", func(t *testing.T) {
		prevType := lookup.target.Type
		lookup.target.Type = assetdomain.TargetTypeIP
		t.Cleanup(func() {
			lookup.target.Type = prevType
		})
		recorder := performSubdomainRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/subdomains:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"dnsNames":["api.example.com"]}`, "application/json")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("batch create success", func(t *testing.T) {
		recorder := performSubdomainRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/subdomains:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"dnsNames":["api.example.com"]}`, "application/json")
		if recorder.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", recorder.Code)
		}
	})

	t.Run("batch create target not found", func(t *testing.T) {
		prevErr := lookup.err
		lookup.err = gorm.ErrRecordNotFound
		t.Cleanup(func() {
			lookup.err = prevErr
		})
		recorder := performSubdomainRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/subdomains:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"dnsNames":["api.example.com"]}`, "application/json")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
	})

	t.Run("batch create internal error", func(t *testing.T) {
		prevErr := store.batchCreateErr
		store.batchCreateErr = fmt.Errorf("create failed")
		t.Cleanup(func() {
			store.batchCreateErr = prevErr
		})
		recorder := performSubdomainRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/subdomains:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"dnsNames":["api.example.com"]}`, "application/json")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
	})

	t.Run("batch delete bind failure", func(t *testing.T) {
		recorder := performSubdomainRequest(t, handler.BatchDelete, http.MethodPost, "/v1/subdomains:batchDelete", nil, "{}", "text/plain")
		if recorder.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("expected 415, got %d", recorder.Code)
		}
	})

	t.Run("batch delete success", func(t *testing.T) {
		recorder := performSubdomainRequest(t, handler.BatchDelete, http.MethodPost, "/v1/subdomains:batchDelete", nil, `{"names":["targets/1/subdomains/1","targets/1/subdomains/2"]}`, "application/json")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
	})

	t.Run("batch delete internal error", func(t *testing.T) {
		prevErr := store.batchDeleteErr
		store.batchDeleteErr = fmt.Errorf("delete failed")
		t.Cleanup(func() {
			store.batchDeleteErr = prevErr
		})
		recorder := performSubdomainRequest(t, handler.BatchDelete, http.MethodPost, "/v1/subdomains:batchDelete", nil, `{"names":["targets/1/subdomains/1","targets/1/subdomains/2"]}`, "application/json")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
	})

	t.Run("export invalid id", func(t *testing.T) {
		recorder := performSubdomainRequest(t, handler.Export, http.MethodGet, "/v1/targets/bad/subdomains/exportFiles/current", gin.Params{{Key: "target", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("export target not found", func(t *testing.T) {
		prevErr := lookup.err
		lookup.err = gorm.ErrRecordNotFound
		t.Cleanup(func() {
			lookup.err = prevErr
		})
		recorder := performSubdomainRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/subdomains/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
	})

	t.Run("export success", func(t *testing.T) {
		recorder := performSubdomainRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/subdomains/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "target_id,dns_name,created_at") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("export stream error", func(t *testing.T) {
		prevErr := store.streamErr
		store.streamErr = fmt.Errorf("stream failed")
		t.Cleanup(func() {
			store.streamErr = prevErr
		})
		recorder := performSubdomainRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/subdomains/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "target_id,dns_name,created_at") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("export count error", func(t *testing.T) {
		prevErr := store.countErr
		store.countErr = fmt.Errorf("count failed")
		t.Cleanup(func() {
			store.countErr = prevErr
		})
		recorder := performSubdomainRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/subdomains/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
	})

	t.Run("export scan error", func(t *testing.T) {
		prevErr := store.scanErr
		store.scanErr = fmt.Errorf("scan failed")
		t.Cleanup(func() {
			store.scanErr = prevErr
		})
		recorder := performSubdomainRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/subdomains/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "target_id,dns_name,created_at") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})
}

func performSubdomainRequest(t *testing.T, handlerFunc gin.HandlerFunc, method, target string, params gin.Params, body, contentType string) *httptest.ResponseRecorder {
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
