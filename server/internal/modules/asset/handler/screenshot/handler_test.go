package screenshot

import (
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

type screenshotStoreStub struct {
	items          []assetdomain.Screenshot
	total          int64
	screenshotByID map[int]*assetdomain.Screenshot
	listErr        error
	getErr         error
	deleteErr      error
	upsertErr      error
	lastPageSize   int
	lastFilter     string
	lastOrderBy    string
	lastOptionField string
}

func (stub *screenshotStoreStub) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Screenshot, int64, error) {
	_ = targetID
	_ = page
	stub.lastPageSize = pageSize
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]assetdomain.Screenshot(nil), stub.items...), stub.total, nil
}

func (stub *screenshotStoreStub) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	_ = targetID
	stub.lastOptionField = field
	return []assetdomain.FilterOption{{Value: "200", Label: "200", Count: 1}}, nil
}

func (stub *screenshotStoreStub) GetByID(id int) (*assetdomain.Screenshot, error) {
	if stub.getErr != nil {
		return nil, stub.getErr
	}
	item, ok := stub.screenshotByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyItem := *item
	return &copyItem, nil
}

func (stub *screenshotStoreStub) BatchDelete(ids []int) (int64, error) {
	if stub.deleteErr != nil {
		return 0, stub.deleteErr
	}
	return int64(len(ids)), nil
}

func (stub *screenshotStoreStub) BatchUpsert(items []assetdomain.Screenshot) (int64, error) {
	if stub.upsertErr != nil {
		return 0, stub.upsertErr
	}
	return int64(len(items)), nil
}

type screenshotLookupStub struct {
	target *assetdomain.TargetRef
	err    error
}

func (stub *screenshotLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	if stub.target == nil || stub.target.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	copyTarget := *stub.target
	return &copyTarget, nil
}

func TestScreenshotHandlerCoverage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	store := &screenshotStoreStub{
		items: []assetdomain.Screenshot{{ID: 1, TargetID: 1, URL: "https://example.com", CreatedAt: now, UpdatedAt: now}},
		total: 1,
		screenshotByID: map[int]*assetdomain.Screenshot{
			1: {ID: 1, TargetID: 1, URL: "https://example.com", Image: []byte("img"), CreatedAt: now, UpdatedAt: now},
			2: {ID: 2, TargetID: 1, URL: "https://example.com/empty", Image: nil, CreatedAt: now, UpdatedAt: now},
		},
	}
	lookup := &screenshotLookupStub{target: &assetdomain.TargetRef{ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}
	handler := NewScreenshotHandler(service.NewScreenshotFacade(service.NewScreenshotQueryService(store, lookup), service.NewScreenshotCommandService(store, lookup)))

	if handler == nil || handler.svc == nil {
		t.Fatal("expected screenshot handler")
	}

	t.Run("list invalid target", func(t *testing.T) {
		recorder := performScreenshotRequest(t, handler.List, http.MethodGet, "/v1/targets/bad/screenshots", gin.Params{{Key: "target", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list invalid query", func(t *testing.T) {
		recorder := performScreenshotRequest(t, handler.List, http.MethodGet, "/v1/targets/1/screenshots?pageSize=abc", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list target not found", func(t *testing.T) {
		lookup.err = gorm.ErrRecordNotFound
		recorder := performScreenshotRequest(t, handler.List, http.MethodGet, "/v1/targets/1/screenshots", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
		lookup.err = nil
	})

	t.Run("list success", func(t *testing.T) {
		recorder := performScreenshotRequest(t, handler.List, http.MethodGet, `/v1/targets/1/screenshots?pageSize=12&filter=url%3D%22admin%22%20%26%26%20statusCode%3D%22200%22&orderBy=statusCode%20desc`, gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "https://example.com") || !strings.Contains(recorder.Body.String(), "totalSize") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
		if store.lastPageSize != 12 || store.lastFilter != `url="admin" && statusCode="200"` || store.lastOrderBy != "statusCode desc" {
			t.Fatalf("unexpected list args size=%d filter=%q orderBy=%q", store.lastPageSize, store.lastFilter, store.lastOrderBy)
		}
	})

	t.Run("list rejects legacy page before store access", func(t *testing.T) {
		store.lastOrderBy = ""
		recorder := performScreenshotRequest(t, handler.List, http.MethodGet, "/v1/targets/1/screenshots?page=2", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
		if store.lastOrderBy != "" {
			t.Fatalf("legacy page must be rejected before store access, got orderBy=%q", store.lastOrderBy)
		}
	})

	t.Run("list rejects unsupported orderBy", func(t *testing.T) {
		recorder := performScreenshotRequest(t, handler.List, http.MethodGet, "/v1/targets/1/screenshots?orderBy=url%20desc", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("filter options success", func(t *testing.T) {
		recorder := performScreenshotRequest(t, handler.FilterOptions, http.MethodGet, "/v1/targets/1/screenshots/filterOptions?field=statusCode", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		body := recorder.Body.String()
		if !strings.Contains(body, `"results"`) || !strings.Contains(body, `"value":"200"`) || !strings.Contains(body, `"count":1`) {
			t.Fatalf("unexpected options response: %s", body)
		}
		if store.lastOptionField != "statusCode" {
			t.Fatalf("expected statusCode option field, got %q", store.lastOptionField)
		}
	})

	t.Run("filter options rejects unsupported field", func(t *testing.T) {
		recorder := performScreenshotRequest(t, handler.FilterOptions, http.MethodGet, "/v1/targets/1/screenshots/filterOptions?field=url", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("list internal error", func(t *testing.T) {
		store.listErr = fmt.Errorf("list failed")
		recorder := performScreenshotRequest(t, handler.List, http.MethodGet, "/v1/targets/1/screenshots", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.listErr = nil
	})

	t.Run("get image invalid id", func(t *testing.T) {
		recorder := performScreenshotRequest(t, handler.GetImage, http.MethodGet, "/v1/screenshots/bad/blob", gin.Params{{Key: "screenshot", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("get image not found", func(t *testing.T) {
		recorder := performScreenshotRequest(t, handler.GetImage, http.MethodGet, "/v1/screenshots/404/blob", gin.Params{{Key: "screenshot", Value: "404"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
	})

	t.Run("get image empty body", func(t *testing.T) {
		recorder := performScreenshotRequest(t, handler.GetImage, http.MethodGet, "/v1/screenshots/2/blob", gin.Params{{Key: "screenshot", Value: "2"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
	})

	t.Run("get image success", func(t *testing.T) {
		recorder := performScreenshotRequest(t, handler.GetImage, http.MethodGet, "/v1/screenshots/1/blob", gin.Params{{Key: "screenshot", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Type") != "image/webp" {
			t.Fatalf("unexpected response: code=%d headers=%v", recorder.Code, recorder.Header())
		}
	})

	t.Run("get image internal error", func(t *testing.T) {
		store.getErr = fmt.Errorf("get failed")
		recorder := performScreenshotRequest(t, handler.GetImage, http.MethodGet, "/v1/screenshots/1/blob", gin.Params{{Key: "screenshot", Value: "1"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.getErr = nil
	})

	t.Run("batch delete bind failure", func(t *testing.T) {
		recorder := performScreenshotRequest(t, handler.BatchDelete, http.MethodPost, "/v1/screenshots:batchDelete", nil, "{}", "text/plain")
		if recorder.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("expected 415, got %d", recorder.Code)
		}
	})

	t.Run("batch delete success", func(t *testing.T) {
		recorder := performScreenshotRequest(t, handler.BatchDelete, http.MethodPost, "/v1/screenshots:batchDelete", nil, `{"names":["targets/1/screenshots/1","targets/1/screenshots/2"]}`, "application/json")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
	})

	t.Run("batch delete internal error", func(t *testing.T) {
		store.deleteErr = fmt.Errorf("delete failed")
		recorder := performScreenshotRequest(t, handler.BatchDelete, http.MethodPost, "/v1/screenshots:batchDelete", nil, `{"names":["targets/1/screenshots/1","targets/1/screenshots/2"]}`, "application/json")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
		store.deleteErr = nil
	})
}

func performScreenshotRequest(t *testing.T, handlerFunc gin.HandlerFunc, method, target string, params gin.Params, body, contentType string) *httptest.ResponseRecorder {
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
