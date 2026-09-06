package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/handler"
)

func TestRegisterScanRoutes_UsesStopCustomMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/v1")

	RegisterScanRoutes(
		protected,
		handler.NewScanHandler(nil, handler.ScanHistoryRetentionPolicy{}),
		handler.NewTaskProgressLogHandler(nil),
	)

	if !hasRoute(engine.Routes(), "POST", "/v1/scans/:scan") {
		t.Fatalf("missing scan stop custom method route")
	}
	if hasRoute(engine.Routes(), "PATCH", "/v1/scans/:scan") {
		t.Fatalf("scan stop must not remain registered as PATCH /v1/scans/:scan")
	}
}

func TestRegisterScanRoutes_DispatchesCanonicalScanCollectionCustomMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/v1")

	RegisterScanRoutes(
		protected,
		handler.NewScanHandler(nil, handler.ScanHistoryRetentionPolicy{}),
		handler.NewTaskProgressLogHandler(nil),
	)

	if !hasRoute(engine.Routes(), "POST", "/v1/scans:customMethod") {
		t.Fatalf("missing scan custom method dispatch route")
	}
	if hasRoute(engine.Routes(), "POST", "/v1/scans") {
		t.Fatalf("scan normal create must use POST /v1/scans:batchCreate, not POST /v1/scans")
	}

	for _, path := range []string{"/v1/scans:batchCreate", "/v1/scans:batchStop", "/v1/scans:quickCreate"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("POST", path, nil)
		engine.ServeHTTP(recorder, request)
		if recorder.Code == http.StatusNotFound {
			t.Fatalf("canonical scan custom method path %s was not dispatched", path)
		}
	}
}

func hasRoute(routes gin.RoutesInfo, method, path string) bool {
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return true
		}
	}
	return false
}
