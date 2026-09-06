package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/handler"
)

func TestRegisterScheduledScanRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/v1")

	RegisterScheduledScanRoutes(protected, handler.NewScheduledScanHandler(nil))

	for _, route := range []struct {
		method string
		path   string
	}{
		{method: "GET", path: "/v1/scheduledScans"},
		{method: "GET", path: "/v1/scheduledScans:summarize"},
		{method: "POST", path: "/v1/scheduledScans"},
		{method: "POST", path: "/v1/scheduledScans:batchUpdate"},
		{method: "GET", path: "/v1/scheduledScans/:scheduled_scan"},
		{method: "PATCH", path: "/v1/scheduledScans/:scheduled_scan"},
		{method: "DELETE", path: "/v1/scheduledScans/:scheduled_scan"},
	} {
		if !hasRoute(engine.Routes(), route.method, route.path) {
			t.Fatalf("missing route %s %s", route.method, route.path)
		}
	}
	if hasRoute(engine.Routes(), "POST", "/v1/scheduledScans/:scheduled_scan") {
		t.Fatal("scheduled scan enabled-state changes must use PATCH with updateMask, not POST toggle")
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
