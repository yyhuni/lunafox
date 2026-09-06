package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/snapshot/handler"
)

func TestRegisterScanSnapshotRoutesIncludesFilterOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/v1")

	RegisterScanSnapshotRoutes(
		protected,
		handler.NewWebsiteSnapshotHandler(nil),
		handler.NewSubdomainSnapshotHandler(nil),
		handler.NewEndpointSnapshotHandler(nil),
		handler.NewDirectorySnapshotHandler(nil),
		handler.NewHostPortSnapshotHandler(nil),
		handler.NewScreenshotSnapshotHandler(nil),
		handler.NewVulnerabilitySnapshotHandler(nil),
	)

	want := []struct {
		method string
		path   string
	}{
		{method: "GET", path: "/v1/scans/:scan/websites/filterOptions"},
		{method: "GET", path: "/v1/scans/:scan/endpoints/filterOptions"},
		{method: "GET", path: "/v1/scans/:scan/directories/filterOptions"},
		{method: "GET", path: "/v1/scans/:scan/hostPorts/filterOptions"},
		{method: "GET", path: "/v1/scans/:scan/screenshots/filterOptions"},
	}

	for _, route := range want {
		if !hasRoute(engine.Routes(), route.method, route.path) {
			t.Fatalf("missing route %s %s", route.method, route.path)
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
