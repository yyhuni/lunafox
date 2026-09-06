package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type websiteSnapshotHandlerStoreStub struct {
	items                  []snapshotdomain.WebsiteSnapshot
	total                  int64
	lastScanID             int
	lastPage               int
	lastPageSize           int
	lastFilter             string
	lastOrderBy            string
	lastFilterOptionsScan  int
	lastFilterOptionsField string
}

func (stub *websiteSnapshotHandlerStoreStub) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.WebsiteSnapshot, int64, error) {
	stub.lastScanID = scanID
	stub.lastPage = page
	stub.lastPageSize = pageSize
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
	return append([]snapshotdomain.WebsiteSnapshot(nil), stub.items...), stub.total, nil
}

func (stub *websiteSnapshotHandlerStoreStub) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	stub.lastFilterOptionsScan = scanID
	stub.lastFilterOptionsField = field
	return []snapshotdomain.FilterOption{{Value: "nginx", Label: "nginx", Count: 1}}, nil
}

func (stub *websiteSnapshotHandlerStoreStub) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.WebsiteSnapshot) error) error {
	return nil
}

func (stub *websiteSnapshotHandlerStoreStub) CountByScanID(scanID int) (int64, error) {
	return stub.total, nil
}

type websiteSnapshotHandlerLookupStub struct{}

func (stub *websiteSnapshotHandlerLookupStub) GetScanRefByID(id int) (*snapshotdomain.ScanRef, error) {
	if id != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	return &snapshotdomain.ScanRef{ID: id}, nil
}

func (stub *websiteSnapshotHandlerLookupStub) GetTargetRefByScanID(scanID int) (*snapshotdomain.ScanTargetRef, error) {
	return nil, gorm.ErrRecordNotFound
}

func TestWebsiteSnapshotHandlerListUsesCanonicalQueryParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
	status := 200
	contentLength := 1024
	store := &websiteSnapshotHandlerStoreStub{
		items: []snapshotdomain.WebsiteSnapshot{{
			ID:            518,
			ScanID:        1,
			URL:           "https://example.com/admin",
			StatusCode:    &status,
			ContentLength: &contentLength,
			CreatedAt:     now,
		}},
		total: 1,
	}
	handler := NewWebsiteSnapshotHandler(service.NewWebsiteSnapshotFacade(
		service.NewWebsiteSnapshotQueryService(store, &websiteSnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performWebsiteSnapshotListRequest(handler, "/v1/scans/1/websites?pageSize=1&filter=url%3D%22admin%22&orderBy=contentLength%20desc")
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "https://example.com/admin") || !strings.Contains(recorder.Body.String(), "totalSize") {
		t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastScanID != 1 || store.lastPage != 1 || store.lastPageSize != 1 || store.lastFilter != `url="admin"` || store.lastOrderBy != "contentLength desc" {
		t.Fatalf("unexpected list args: scan=%d page=%d pageSize=%d filter=%q orderBy=%q", store.lastScanID, store.lastPage, store.lastPageSize, store.lastFilter, store.lastOrderBy)
	}
}

func TestWebsiteSnapshotHandlerListRejectsLegacyPageBeforeStoreAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &websiteSnapshotHandlerStoreStub{}
	handler := NewWebsiteSnapshotHandler(service.NewWebsiteSnapshotFacade(
		service.NewWebsiteSnapshotQueryService(store, &websiteSnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performWebsiteSnapshotListRequest(handler, "/v1/scans/1/websites?page=2")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastScanID != 0 {
		t.Fatal("legacy page parameter must be rejected before store access")
	}
}

func TestWebsiteSnapshotHandlerListRejectsUnsupportedOrderBy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &websiteSnapshotHandlerStoreStub{}
	handler := NewWebsiteSnapshotHandler(service.NewWebsiteSnapshotFacade(
		service.NewWebsiteSnapshotQueryService(store, &websiteSnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performWebsiteSnapshotListRequest(handler, "/v1/scans/1/websites?orderBy=tech%20desc")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestWebsiteSnapshotHandlerListRejectsURLOrderByBeforeStoreAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &websiteSnapshotHandlerStoreStub{}
	handler := NewWebsiteSnapshotHandler(service.NewWebsiteSnapshotFacade(
		service.NewWebsiteSnapshotQueryService(store, &websiteSnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performWebsiteSnapshotListRequest(handler, "/v1/scans/1/websites?orderBy=url%20desc")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastScanID != 0 {
		t.Fatalf("url orderBy must be rejected before store access, got scan=%d", store.lastScanID)
	}
}

func TestWebsiteSnapshotHandlerFilterOptionsUsesScanScopedResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &websiteSnapshotHandlerStoreStub{}
	handler := NewWebsiteSnapshotHandler(service.NewWebsiteSnapshotFacade(
		service.NewWebsiteSnapshotQueryService(store, &websiteSnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performWebsiteSnapshotFilterOptionsRequest(handler, "/v1/scans/1/websites/filterOptions?field=tech")
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"results"`) || !strings.Contains(recorder.Body.String(), `"value":"nginx"`) || !strings.Contains(recorder.Body.String(), `"count":1`) {
		t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastFilterOptionsScan != 1 || store.lastFilterOptionsField != "tech" {
		t.Fatalf("unexpected filter option args: scan=%d field=%q", store.lastFilterOptionsScan, store.lastFilterOptionsField)
	}
}

func TestWebsiteSnapshotHandlerFilterOptionsRejectsUnsupportedField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &websiteSnapshotHandlerStoreStub{}
	handler := NewWebsiteSnapshotHandler(service.NewWebsiteSnapshotFacade(
		service.NewWebsiteSnapshotQueryService(store, &websiteSnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performWebsiteSnapshotFilterOptionsRequest(handler, "/v1/scans/1/websites/filterOptions?field=responseBody")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func performWebsiteSnapshotListRequest(handler *WebsiteSnapshotHandler, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	c.Params = gin.Params{{Key: "scan", Value: "1"}}
	handler.List(c)
	return recorder
}

func performWebsiteSnapshotFilterOptionsRequest(handler *WebsiteSnapshotHandler, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	c.Params = gin.Params{{Key: "scan", Value: "1"}}
	handler.FilterOptions(c)
	return recorder
}
