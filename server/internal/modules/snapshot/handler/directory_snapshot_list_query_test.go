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

type directorySnapshotHandlerStoreStub struct {
	items                  []snapshotdomain.DirectorySnapshot
	total                  int64
	lastScanID             int
	lastPage               int
	lastPageSize           int
	lastFilter             string
	lastOrderBy            string
	lastFilterOptionsScan  int
	lastFilterOptionsField string
}

func (stub *directorySnapshotHandlerStoreStub) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.DirectorySnapshot, int64, error) {
	stub.lastScanID = scanID
	stub.lastPage = page
	stub.lastPageSize = pageSize
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
	return append([]snapshotdomain.DirectorySnapshot(nil), stub.items...), stub.total, nil
}

func (stub *directorySnapshotHandlerStoreStub) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	stub.lastFilterOptionsScan = scanID
	stub.lastFilterOptionsField = field
	return []snapshotdomain.FilterOption{{Value: "text/html", Label: "text/html", Count: 1}}, nil
}

func (stub *directorySnapshotHandlerStoreStub) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.DirectorySnapshot) error) error {
	for _, item := range stub.items {
		if err := visit(item); err != nil {
			return err
		}
	}
	return nil
}

func (stub *directorySnapshotHandlerStoreStub) CountByScanID(scanID int) (int64, error) {
	return stub.total, nil
}

type directorySnapshotHandlerLookupStub struct{}

func (stub *directorySnapshotHandlerLookupStub) GetScanRefByID(id int) (*snapshotdomain.ScanRef, error) {
	if id != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	return &snapshotdomain.ScanRef{ID: id}, nil
}

func (stub *directorySnapshotHandlerLookupStub) GetTargetRefByScanID(scanID int) (*snapshotdomain.ScanTargetRef, error) {
	return nil, gorm.ErrRecordNotFound
}

func TestDirectorySnapshotHandlerListUsesCanonicalQueryParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
	status := 200
	contentLength := int64(1024)
	store := &directorySnapshotHandlerStoreStub{
		items: []snapshotdomain.DirectorySnapshot{{
			ID:            518,
			ScanID:        1,
			URL:           "https://example.com/admin",
			Status:        &status,
			ContentLength: &contentLength,
			ContentType:   "text/html",
			CreatedAt:     now,
		}},
		total: 1,
	}
	handler := NewDirectorySnapshotHandler(service.NewDirectorySnapshotFacade(
		service.NewDirectorySnapshotQueryService(store, &directorySnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performDirectorySnapshotListRequest(handler, "/v1/scans/1/directories?pageSize=1&filter=url%3D%22admin%22&orderBy=contentLength%20desc")
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "https://example.com/admin") || !strings.Contains(recorder.Body.String(), "totalSize") {
		t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastScanID != 1 || store.lastPage != 1 || store.lastPageSize != 1 || store.lastFilter != `url="admin"` || store.lastOrderBy != "contentLength desc" {
		t.Fatalf("unexpected list args: scan=%d page=%d pageSize=%d filter=%q orderBy=%q", store.lastScanID, store.lastPage, store.lastPageSize, store.lastFilter, store.lastOrderBy)
	}
}

func TestDirectorySnapshotHandlerListRejectsLegacyPageBeforeStoreAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &directorySnapshotHandlerStoreStub{}
	handler := NewDirectorySnapshotHandler(service.NewDirectorySnapshotFacade(
		service.NewDirectorySnapshotQueryService(store, &directorySnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performDirectorySnapshotListRequest(handler, "/v1/scans/1/directories?page=2")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastScanID != 0 {
		t.Fatal("legacy page parameter must be rejected before store access")
	}
}

func TestDirectorySnapshotHandlerListRejectsUnsupportedOrderBy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &directorySnapshotHandlerStoreStub{}
	handler := NewDirectorySnapshotHandler(service.NewDirectorySnapshotFacade(
		service.NewDirectorySnapshotQueryService(store, &directorySnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performDirectorySnapshotListRequest(handler, "/v1/scans/1/directories?orderBy=url%20desc")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastScanID != 0 {
		t.Fatalf("url orderBy must be rejected before store access, got scan=%d", store.lastScanID)
	}
}

func TestDirectorySnapshotHandlerFilterOptionsUsesScanScopedResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &directorySnapshotHandlerStoreStub{}
	handler := NewDirectorySnapshotHandler(service.NewDirectorySnapshotFacade(
		service.NewDirectorySnapshotQueryService(store, &directorySnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performDirectorySnapshotFilterOptionsRequest(handler, "/v1/scans/1/directories/filterOptions?field=contentType")
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"results"`) || !strings.Contains(recorder.Body.String(), `"value":"text/html"`) || !strings.Contains(recorder.Body.String(), `"count":1`) {
		t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastFilterOptionsScan != 1 || store.lastFilterOptionsField != "contentType" {
		t.Fatalf("unexpected filter option args: scan=%d field=%q", store.lastFilterOptionsScan, store.lastFilterOptionsField)
	}
}

func TestDirectorySnapshotHandlerFilterOptionsRejectsUnsupportedField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &directorySnapshotHandlerStoreStub{}
	handler := NewDirectorySnapshotHandler(service.NewDirectorySnapshotFacade(
		service.NewDirectorySnapshotQueryService(store, &directorySnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performDirectorySnapshotFilterOptionsRequest(handler, "/v1/scans/1/directories/filterOptions?field=url")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func performDirectorySnapshotListRequest(handler *DirectorySnapshotHandler, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	c.Params = gin.Params{{Key: "scan", Value: "1"}}
	handler.List(c)
	return recorder
}

func performDirectorySnapshotFilterOptionsRequest(handler *DirectorySnapshotHandler, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	c.Params = gin.Params{{Key: "scan", Value: "1"}}
	handler.FilterOptions(c)
	return recorder
}
