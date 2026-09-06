package website

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

type websiteStoreStub struct {
	items           []assetdomain.Website
	total           int64
	count           int64
	websiteByID     map[int]*assetdomain.Website
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
	lastPageSize    int
	lastFilter      string
	lastOrderBy     string
	lastOptionField string
	screenshots     []assetdomain.Screenshot
	screenshotErr   error
}

func (stub *websiteStoreStub) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Website, int64, error) {
	_ = targetID
	_ = page
	stub.lastPageSize = pageSize
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]assetdomain.Website(nil), stub.items...), stub.total, nil
}

func (stub *websiteStoreStub) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	_ = targetID
	stub.lastOptionField = field
	return []assetdomain.FilterOption{{Value: "gin", Label: "gin", Count: 1}}, nil
}

func (stub *websiteStoreStub) ListSummariesByTargetAndURLs(targetID int, urls []string) ([]assetdomain.Screenshot, error) {
	_ = targetID
	_ = urls
	if stub.screenshotErr != nil {
		return nil, stub.screenshotErr
	}
	return append([]assetdomain.Screenshot(nil), stub.screenshots...), nil
}

func (stub *websiteStoreStub) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Website) error) error {
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

func (stub *websiteStoreStub) CountByTargetID(targetID int) (int64, error) {
	_ = targetID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func (stub *websiteStoreStub) GetByID(id int) (*assetdomain.Website, error) {
	if stub.getErr != nil {
		return nil, stub.getErr
	}
	item, ok := stub.websiteByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyItem := *item
	return &copyItem, nil
}

func (stub *websiteStoreStub) BatchCreate(websites []assetdomain.Website) (int, error) {
	if stub.batchCreateErr != nil {
		return 0, stub.batchCreateErr
	}
	return len(websites), nil
}

func (stub *websiteStoreStub) BatchCreateContext(_ context.Context, websites []assetdomain.Website) (int, error) {
	return stub.BatchCreate(websites)
}

func (stub *websiteStoreStub) Delete(id int) error {
	if stub.deleteErr != nil {
		return stub.deleteErr
	}
	stub.deletedID = id
	return nil
}

func (stub *websiteStoreStub) BatchDelete(ids []int) (int64, error) {
	if stub.batchDeleteErr != nil {
		return 0, stub.batchDeleteErr
	}
	stub.batchDeletedIDs = append([]int(nil), ids...)
	return int64(len(ids)), nil
}

func (stub *websiteStoreStub) BatchUpsert(websites []assetdomain.Website) (int64, error) {
	if stub.batchUpsertErr != nil {
		return 0, stub.batchUpsertErr
	}
	return int64(len(websites)), nil
}

func (stub *websiteStoreStub) BatchUpsertContext(_ context.Context, websites []assetdomain.Website) (int64, error) {
	return stub.BatchUpsert(websites)
}

func (stub *websiteStoreStub) BatchUpsertTechnologyContext(_ context.Context, _ int, websites []assetdomain.WebsiteTechnology) (int64, error) {
	return int64(len(websites)), nil
}

type websiteLookupStub struct {
	target *assetdomain.TargetRef
	err    error
}

func (stub *websiteLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	if stub.target == nil || stub.target.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	copyTarget := *stub.target
	return &copyTarget, nil
}

func (stub *websiteLookupStub) GetActiveByIDContext(ctx context.Context, id int) (*assetdomain.TargetRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.GetActiveByID(id)
}

func TestWebsiteHandlerNewListExportAndWrites(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	status := 200
	screenshotStatus := int16(200)
	contentLength := 512
	vhost := true
	store := &websiteStoreStub{
		items: []assetdomain.Website{{
			ID:              1,
			TargetID:        1,
			URL:             "https://example.com",
			Host:            "example.com",
			Title:           "home",
			StatusCode:      &status,
			ContentLength:   &contentLength,
			Tech:            []string{"go", "gin"},
			Vhost:           &vhost,
			ResponseHeaders: "server: nginx",
			CreatedAt:       now,
		}},
		total: 1,
		count: 1,
		websiteByID: map[int]*assetdomain.Website{
			2: {ID: 2, TargetID: 1, URL: "https://example.com/profile", Host: "example.com"},
		},
		screenshots: []assetdomain.Screenshot{{
			ID:         9,
			TargetID:   1,
			URL:        "https://example.com/profile",
			StatusCode: &screenshotStatus,
			Image:      []byte("must-not-be-in-website-output"),
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
	}
	lookup := &websiteLookupStub{target: &assetdomain.TargetRef{ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}
	handler := NewWebsiteHandler(service.NewWebsiteFacade(service.NewWebsiteQueryService(store, store, lookup), service.NewWebsiteCommandService(store, lookup)))

	if handler == nil || handler.svc == nil {
		t.Fatal("expected website handler to be initialized")
	}

	t.Run("get invalid id", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.Get, http.MethodGet, "/v1/websites/bad", gin.Params{{Key: "website", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.Get, http.MethodGet, "/v1/websites/404", gin.Params{{Key: "website", Value: "404"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
	})

	t.Run("get success includes screenshot summary only", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.Get, http.MethodGet, "/v1/websites/2", gin.Params{{Key: "website", Value: "2"}}, "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		body := recorder.Body.String()
		for _, expected := range []string{`"name":"targets/1/websites/2"`, `"screenshot"`, `"name":"targets/1/screenshots/9"`} {
			if !strings.Contains(body, expected) {
				t.Fatalf("expected %q in response: %s", expected, body)
			}
		}
		if strings.Contains(body, "must-not-be-in-website-output") || strings.Contains(body, `"image"`) {
			t.Fatalf("website output must not contain screenshot bytes: %s", body)
		}
	})

	t.Run("list invalid target", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.List, http.MethodGet, "/v1/targets/bad/websites", gin.Params{{Key: "target", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list invalid query", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.List, http.MethodGet, "/v1/targets/1/websites?pageSize=abc", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list internal error", func(t *testing.T) {
		prevErr := store.listErr
		store.listErr = fmt.Errorf("list failed")
		t.Cleanup(func() {
			store.listErr = prevErr
		})
		recorder := performWebsiteRequest(t, handler.List, http.MethodGet, "/v1/targets/1/websites", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
	})

	t.Run("list target not found", func(t *testing.T) {
		prevErr := store.listErr
		store.listErr = service.ErrTargetNotFound
		t.Cleanup(func() {
			store.listErr = prevErr
		})
		recorder := performWebsiteRequest(t, handler.List, http.MethodGet, "/v1/targets/1/websites", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
	})

	t.Run("list success", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.List, http.MethodGet, `/v1/targets/1/websites?pageSize=20&filter=url%3D%22admin%22&orderBy=contentLength%20desc`, gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		body := recorder.Body.String()
		if !strings.Contains(body, "\"results\"") || !strings.Contains(body, "\"totalSize\":1") || !strings.Contains(body, "https://example.com") {
			t.Fatalf("unexpected body: %s", body)
		}
		if store.lastPageSize != 20 || store.lastFilter != `url="admin"` || store.lastOrderBy != "contentLength desc" {
			t.Fatalf("unexpected list args size=%d filter=%q orderBy=%q", store.lastPageSize, store.lastFilter, store.lastOrderBy)
		}
	})

	t.Run("list rejects legacy page", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.List, http.MethodGet, "/v1/targets/1/websites?page=2", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list rejects unsupported orderBy", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.List, http.MethodGet, "/v1/targets/1/websites?orderBy=tech%20desc", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("list rejects url orderBy before store access", func(t *testing.T) {
		store.lastOrderBy = ""
		recorder := performWebsiteRequest(t, handler.List, http.MethodGet, "/v1/targets/1/websites?orderBy=url%20desc", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
		if store.lastOrderBy != "" {
			t.Fatalf("url orderBy must be rejected before store access, got orderBy=%q", store.lastOrderBy)
		}
	})

	t.Run("filter options success", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.FilterOptions, http.MethodGet, "/v1/targets/1/websites/filterOptions?field=tech", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		body := recorder.Body.String()
		if !strings.Contains(body, `"results"`) || !strings.Contains(body, `"value":"gin"`) || !strings.Contains(body, `"label":"gin"`) || !strings.Contains(body, `"count":1`) {
			t.Fatalf("unexpected options response: %s", body)
		}
	})

	t.Run("filter options rejects unsupported field", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.FilterOptions, http.MethodGet, "/v1/targets/1/websites/filterOptions?field=url", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("batch create bind failure", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/websites:batchCreate", gin.Params{{Key: "target", Value: "1"}}, "{}", "text/plain")
		if recorder.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("expected 415, got %d", recorder.Code)
		}
	})

	t.Run("batch create invalid target id", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/bad/websites:batchCreate", gin.Params{{Key: "target", Value: "bad"}}, `{"urls":["https://example.com"]}`, "application/json")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("batch create invalid url is a client error", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/websites:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"urls":["https:%zz"]}`, "application/json")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("batch create not found", func(t *testing.T) {
		prevErr := lookup.err
		lookup.err = gorm.ErrRecordNotFound
		t.Cleanup(func() {
			lookup.err = prevErr
		})
		recorder := performWebsiteRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/websites:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"urls":["https://example.com"]}`, "application/json")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
	})

	t.Run("batch create success", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/websites:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"urls":["https://example.com"]}`, "application/json")
		if recorder.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", recorder.Code)
		}
		if !strings.Contains(recorder.Body.String(), "\"createdCount\":1") {
			t.Fatalf("unexpected body: %s", recorder.Body.String())
		}
	})

	t.Run("batch create internal error", func(t *testing.T) {
		prevErr := store.batchCreateErr
		store.batchCreateErr = fmt.Errorf("create failed")
		t.Cleanup(func() {
			store.batchCreateErr = prevErr
		})
		recorder := performWebsiteRequest(t, handler.BatchCreate, http.MethodPost, "/v1/targets/1/websites:batchCreate", gin.Params{{Key: "target", Value: "1"}}, `{"urls":["https://example.com"]}`, "application/json")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
	})

	t.Run("delete invalid id", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.Delete, http.MethodDelete, "/v1/websites/bad", gin.Params{{Key: "website", Value: "bad"}}, "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})

	t.Run("delete not found", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.Delete, http.MethodDelete, "/v1/websites/404", gin.Params{{Key: "website", Value: "404"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
	})

	t.Run("delete success", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.Delete, http.MethodDelete, "/v1/websites/2", gin.Params{{Key: "website", Value: "2"}}, "", "")
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", recorder.Code)
		}
		if store.deletedID != 2 {
			t.Fatalf("expected deleted id 2, got %d", store.deletedID)
		}
	})

	t.Run("delete internal error", func(t *testing.T) {
		prevErr := store.getErr
		store.getErr = fmt.Errorf("get failed")
		t.Cleanup(func() {
			store.getErr = prevErr
		})
		recorder := performWebsiteRequest(t, handler.Delete, http.MethodDelete, "/v1/websites/2", gin.Params{{Key: "website", Value: "2"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
	})

	t.Run("batch delete bind failure", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.BatchDelete, http.MethodPost, "/v1/websites:batchDelete", nil, "{}", "text/plain")
		if recorder.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("expected 415, got %d", recorder.Code)
		}
	})

	t.Run("batch delete success", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.BatchDelete, http.MethodPost, "/v1/websites:batchDelete", nil, `{"names":["targets/1/websites/1","targets/1/websites/2"]}`, "application/json")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		if !strings.Contains(recorder.Body.String(), "\"deletedCount\":2") {
			t.Fatalf("unexpected body: %s", recorder.Body.String())
		}
	})

	t.Run("batch delete internal error", func(t *testing.T) {
		prevErr := store.batchDeleteErr
		store.batchDeleteErr = fmt.Errorf("delete failed")
		t.Cleanup(func() {
			store.batchDeleteErr = prevErr
		})
		recorder := performWebsiteRequest(t, handler.BatchDelete, http.MethodPost, "/v1/websites:batchDelete", nil, `{"names":["targets/1/websites/1","targets/1/websites/2"]}`, "application/json")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
	})

	t.Run("export invalid id", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.Export, http.MethodGet, "/v1/targets/bad/websites/exportFiles/current", gin.Params{{Key: "target", Value: "bad"}}, "", "")
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
		recorder := performWebsiteRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/websites/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recorder.Code)
		}
	})

	t.Run("export count error", func(t *testing.T) {
		prevErr := store.countErr
		store.countErr = fmt.Errorf("count failed")
		t.Cleanup(func() {
			store.countErr = prevErr
		})
		recorder := performWebsiteRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/websites/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", recorder.Code)
		}
	})

	t.Run("export success", func(t *testing.T) {
		recorder := performWebsiteRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/websites/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		body := recorder.Body.String()
		if !strings.Contains(body, "target_id,url,host") || !strings.Contains(body, "go|gin") {
			t.Fatalf("unexpected csv body: %s", body)
		}
	})

	t.Run("export stream error", func(t *testing.T) {
		prevErr := store.streamErr
		store.streamErr = fmt.Errorf("stream failed")
		t.Cleanup(func() {
			store.streamErr = prevErr
		})
		recorder := performWebsiteRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/websites/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "target_id,url,host") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("export scan error", func(t *testing.T) {
		prevErr := store.scanErr
		store.scanErr = fmt.Errorf("scan failed")
		t.Cleanup(func() {
			store.scanErr = prevErr
		})
		recorder := performWebsiteRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/websites/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "target_id,url,host") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})
}

func performWebsiteRequest(t *testing.T, handlerFunc gin.HandlerFunc, method, target string, params gin.Params, body, contentType string) *httptest.ResponseRecorder {
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
