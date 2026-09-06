package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type hostPortSnapshotHandlerStoreStub struct {
	lastOptionsScanID int
}

func (stub *hostPortSnapshotHandlerStoreStub) GetIPAggregation(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.HostPortIPAggregationRow, int64, error) {
	return nil, 0, nil
}

func (stub *hostPortSnapshotHandlerStoreStub) GetHostsAndPortsByIP(scanID int, ip string, filter string) ([]string, []int, error) {
	return nil, nil, nil
}

func (stub *hostPortSnapshotHandlerStoreStub) ListPortOptionsByScanID(scanID int) ([]snapshotdomain.FilterOption, error) {
	stub.lastOptionsScanID = scanID
	return []snapshotdomain.FilterOption{{Value: "443", Label: "443", Count: 2}}, nil
}

func (stub *hostPortSnapshotHandlerStoreStub) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.HostPortSnapshot) error) error {
	return nil
}

func (stub *hostPortSnapshotHandlerStoreStub) CountByScanID(scanID int) (int64, error) {
	return 0, nil
}

type hostPortSnapshotHandlerLookupStub struct{}

func (stub *hostPortSnapshotHandlerLookupStub) GetScanRefByID(id int) (*snapshotdomain.ScanRef, error) {
	if id != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	return &snapshotdomain.ScanRef{ID: id}, nil
}

func (stub *hostPortSnapshotHandlerLookupStub) GetTargetRefByScanID(scanID int) (*snapshotdomain.ScanTargetRef, error) {
	return nil, gorm.ErrRecordNotFound
}

func TestHostPortSnapshotHandlerFilterOptionsUsesScanScopedPortResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &hostPortSnapshotHandlerStoreStub{}
	handler := NewHostPortSnapshotHandler(service.NewHostPortSnapshotFacade(
		service.NewHostPortSnapshotQueryService(store, &hostPortSnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performHostPortSnapshotFilterOptionsRequest(handler, "/v1/scans/1/hostPorts/filterOptions?field=port")
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"results"`) || !strings.Contains(recorder.Body.String(), `"value":"443"`) || !strings.Contains(recorder.Body.String(), `"count":2`) {
		t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastOptionsScanID != 1 {
		t.Fatalf("unexpected scan ID for port options: %d", store.lastOptionsScanID)
	}
}

func TestHostPortSnapshotHandlerFilterOptionsRejectsUnsupportedField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHostPortSnapshotHandler(service.NewHostPortSnapshotFacade(
		service.NewHostPortSnapshotQueryService(&hostPortSnapshotHandlerStoreStub{}, &hostPortSnapshotHandlerLookupStub{}),
		nil,
	))

	recorder := performHostPortSnapshotFilterOptionsRequest(handler, "/v1/scans/1/hostPorts/filterOptions?field=host")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func performHostPortSnapshotFilterOptionsRequest(handler *HostPortSnapshotHandler, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	c.Params = gin.Params{{Key: "scan", Value: "1"}}
	handler.FilterOptions(c)
	return recorder
}
