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

type screenshotSnapshotHandlerStoreStub struct {
	items                  []snapshotdomain.ScreenshotSnapshot
	total                  int64
	lastScanID             int
	lastPage               int
	lastPageSize           int
	lastFilter             string
	lastOrderBy            string
	lastFilterOptionsScan  int
	lastFilterOptionsField string
}

func (stub *screenshotSnapshotHandlerStoreStub) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.ScreenshotSnapshot, int64, error) {
	stub.lastScanID = scanID
	stub.lastPage = page
	stub.lastPageSize = pageSize
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
	return append([]snapshotdomain.ScreenshotSnapshot(nil), stub.items...), stub.total, nil
}

func (stub *screenshotSnapshotHandlerStoreStub) ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	stub.lastFilterOptionsScan = scanID
	stub.lastFilterOptionsField = field
	return []snapshotdomain.FilterOption{{Value: "200", Label: "200", Count: 1}}, nil
}

func (stub *screenshotSnapshotHandlerStoreStub) FindByIDAndScanID(id int, scanID int) (*snapshotdomain.ScreenshotSnapshot, error) {
	return nil, gorm.ErrRecordNotFound
}

func (stub *screenshotSnapshotHandlerStoreStub) BatchUpsertContext(_ context.Context, snapshots []snapshotdomain.ScreenshotSnapshot) (int64, error) {
	return int64(len(snapshots)), nil
}

type screenshotSnapshotHandlerLookupStub struct{}

func (stub *screenshotSnapshotHandlerLookupStub) GetScanRefByID(id int) (*snapshotdomain.ScanRef, error) {
	if id != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	return &snapshotdomain.ScanRef{ID: id, TargetID: 7}, nil
}

func (stub *screenshotSnapshotHandlerLookupStub) GetTargetRefByScanID(scanID int) (*snapshotdomain.ScanTargetRef, error) {
	return nil, gorm.ErrRecordNotFound
}

func (stub *screenshotSnapshotHandlerLookupStub) GetScanRefByIDContext(ctx context.Context, id int) (*snapshotdomain.ScanRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.GetScanRefByID(id)
}

func (stub *screenshotSnapshotHandlerLookupStub) GetTargetRefByScanIDContext(ctx context.Context, scanID int) (*snapshotdomain.ScanTargetRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.GetTargetRefByScanID(scanID)
}

type screenshotSnapshotAssetSyncStub struct{}

func (stub *screenshotSnapshotAssetSyncStub) BatchUpsertContext(_ context.Context, targetID int, req *service.ScreenshotAssetUpsertRequest) (int64, error) {
	_ = targetID
	return int64(len(req.Screenshots)), nil
}

func TestScreenshotSnapshotHandlerListUsesCanonicalQueryParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
	status := int16(200)
	store := &screenshotSnapshotHandlerStoreStub{
		items: []snapshotdomain.ScreenshotSnapshot{{
			ID:         518,
			ScanID:     1,
			URL:        "https://example.com/admin",
			StatusCode: &status,
			CreatedAt:  now,
		}},
		total: 1,
	}
	handler := NewScreenshotSnapshotHandler(newScreenshotSnapshotHandlerFacade(store))

	recorder := performScreenshotSnapshotListRequest(handler, "/v1/scans/1/screenshots?pageSize=1&filter=url%3D%22admin%22&orderBy=statusCode%20desc")
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "https://example.com/admin") || !strings.Contains(recorder.Body.String(), "totalSize") {
		t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastScanID != 1 || store.lastPage != 1 || store.lastPageSize != 1 || store.lastFilter != `url="admin"` || store.lastOrderBy != "statusCode desc" {
		t.Fatalf("unexpected list args: scan=%d page=%d pageSize=%d filter=%q orderBy=%q", store.lastScanID, store.lastPage, store.lastPageSize, store.lastFilter, store.lastOrderBy)
	}
}

func TestScreenshotSnapshotHandlerListRejectsLegacyPageBeforeStoreAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &screenshotSnapshotHandlerStoreStub{}
	handler := NewScreenshotSnapshotHandler(newScreenshotSnapshotHandlerFacade(store))

	recorder := performScreenshotSnapshotListRequest(handler, "/v1/scans/1/screenshots?page=2")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastScanID != 0 {
		t.Fatal("legacy page parameter must be rejected before store access")
	}
}

func TestScreenshotSnapshotHandlerListRejectsUnsupportedOrderBy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &screenshotSnapshotHandlerStoreStub{}
	handler := NewScreenshotSnapshotHandler(newScreenshotSnapshotHandlerFacade(store))

	recorder := performScreenshotSnapshotListRequest(handler, "/v1/scans/1/screenshots?orderBy=url%20desc")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestScreenshotSnapshotHandlerFilterOptionsUsesScanScopedResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &screenshotSnapshotHandlerStoreStub{}
	handler := NewScreenshotSnapshotHandler(newScreenshotSnapshotHandlerFacade(store))

	recorder := performScreenshotSnapshotFilterOptionsRequest(handler, "/v1/scans/1/screenshots/filterOptions?field=statusCode")
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"results"`) || !strings.Contains(recorder.Body.String(), `"value":"200"`) || !strings.Contains(recorder.Body.String(), `"count":1`) {
		t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastFilterOptionsScan != 1 || store.lastFilterOptionsField != "statusCode" {
		t.Fatalf("unexpected filter option args: scan=%d field=%q", store.lastFilterOptionsScan, store.lastFilterOptionsField)
	}
}

func TestScreenshotSnapshotHandlerFilterOptionsRejectsUnsupportedField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &screenshotSnapshotHandlerStoreStub{}
	handler := NewScreenshotSnapshotHandler(newScreenshotSnapshotHandlerFacade(store))

	recorder := performScreenshotSnapshotFilterOptionsRequest(handler, "/v1/scans/1/screenshots/filterOptions?field=url")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func newScreenshotSnapshotHandlerFacade(store *screenshotSnapshotHandlerStoreStub) *service.ScreenshotSnapshotFacade {
	lookup := &screenshotSnapshotHandlerLookupStub{}
	return service.NewScreenshotSnapshotFacade(
		service.NewScreenshotSnapshotQueryService(store, lookup),
		service.NewScreenshotSnapshotCommandService(store, lookup, &screenshotSnapshotAssetSyncStub{}),
	)
}

func performScreenshotSnapshotListRequest(handler *ScreenshotSnapshotHandler, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	c.Params = gin.Params{{Key: "scan", Value: "1"}}
	handler.List(c)
	return recorder
}

func performScreenshotSnapshotFilterOptionsRequest(handler *ScreenshotSnapshotHandler, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	c.Params = gin.Params{{Key: "scan", Value: "1"}}
	handler.FilterOptions(c)
	return recorder
}
