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

type subdomainSnapshotHandlerStoreStub struct {
	items        []snapshotdomain.SubdomainSnapshot
	total        int64
	lastScanID   int
	lastPage     int
	lastPageSize int
	lastFilter   string
	lastOrderBy  string
}

func (stub *subdomainSnapshotHandlerStoreStub) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.SubdomainSnapshot, int64, error) {
	stub.lastScanID = scanID
	stub.lastPage = page
	stub.lastPageSize = pageSize
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
	return append([]snapshotdomain.SubdomainSnapshot(nil), stub.items...), stub.total, nil
}

func (stub *subdomainSnapshotHandlerStoreStub) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.SubdomainSnapshot) error) error {
	return nil
}

func (stub *subdomainSnapshotHandlerStoreStub) CountByScanID(scanID int) (int64, error) {
	return stub.total, nil
}

type subdomainSnapshotHandlerLookupStub struct{}

func (stub *subdomainSnapshotHandlerLookupStub) GetScanRefByID(id int) (*snapshotdomain.ScanRef, error) {
	if id != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	return &snapshotdomain.ScanRef{ID: id}, nil
}

func (stub *subdomainSnapshotHandlerLookupStub) GetTargetRefByScanID(scanID int) (*snapshotdomain.ScanTargetRef, error) {
	return nil, gorm.ErrRecordNotFound
}

func TestSubdomainSnapshotHandlerListUsesCanonicalQueryParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	store := &subdomainSnapshotHandlerStoreStub{
		items: []snapshotdomain.SubdomainSnapshot{{ID: 3266, ScanID: 1, DNSName: "api.example.com", CreatedAt: now}},
		total: 1,
	}
	handler := NewSubdomainSnapshotHandler(service.NewSubdomainSnapshotFacade(
		service.NewSubdomainSnapshotQueryService(store, &subdomainSnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performSubdomainSnapshotListRequest(handler, "/v1/scans/1/subdomains?pageSize=1&filter=dnsName%3D%22api%22&orderBy=dnsName%20desc")
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "api.example.com") || !strings.Contains(recorder.Body.String(), "totalSize") {
		t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastScanID != 1 || store.lastPage != 1 || store.lastPageSize != 1 || store.lastFilter != `dnsName="api"` || store.lastOrderBy != "dnsName desc" {
		t.Fatalf("unexpected list args: scan=%d page=%d pageSize=%d filter=%q orderBy=%q", store.lastScanID, store.lastPage, store.lastPageSize, store.lastFilter, store.lastOrderBy)
	}
}

func TestSubdomainSnapshotHandlerListRejectsLegacyPageBeforeStoreAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &subdomainSnapshotHandlerStoreStub{}
	handler := NewSubdomainSnapshotHandler(service.NewSubdomainSnapshotFacade(
		service.NewSubdomainSnapshotQueryService(store, &subdomainSnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performSubdomainSnapshotListRequest(handler, "/v1/scans/1/subdomains?page=2")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastScanID != 0 {
		t.Fatal("legacy page parameter must be rejected before store access")
	}
}

func performSubdomainSnapshotListRequest(handler *SubdomainSnapshotHandler, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	c.Params = gin.Params{{Key: "scan", Value: "1"}}
	handler.List(c)
	return recorder
}
