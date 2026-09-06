package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	assethandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler"
	directoryhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/directory"
	endpointhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/endpoint"
	hostporthandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/host_port"
	screenshothandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/screenshot"
	searchhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/search"
	subdomainhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/subdomain"
	websitehandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/website"
)

func TestRegisterAssetRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/v1")
	protected := engine.Group("/v1")

	RegisterAssetRoutes(
		api,
		protected,
		screenshothandler.NewScreenshotHandler(nil),
		nil,
		websitehandler.NewWebsiteHandler(nil),
		subdomainhandler.NewSubdomainHandler(nil),
		endpointhandler.NewEndpointHandler(nil),
		directoryhandler.NewDirectoryHandler(nil),
		hostporthandler.NewHostPortHandler(nil),
		searchhandler.NewGlobalAssetSearchHandler(nil),
		assethandler.NewAssetStatisticsHandler(nil),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	want := []struct {
		method string
		path   string
	}{
		{method: "GET", path: "/v1/screenshots/:screenshot/blob"},
		{method: "GET", path: "/v1/targets/:target/websites"},
		{method: "GET", path: "/v1/assets:search"},
		{method: "GET", path: "/v1/assetStatistics"},
		{method: "GET", path: "/v1/assetStatistics/history"},
		{method: "GET", path: "/v1/websites/:website"},
		{method: "GET", path: "/v1/targets/:target/websites/filterOptions"},
		{method: "POST", path: "/v1/targets/:target/websites:batchMethod"},
		{method: "GET", path: "/v1/targets/:target/subdomains/exportFiles/current"},
		{method: "GET", path: "/v1/targets/:target/endpoints"},
		{method: "GET", path: "/v1/endpoints/:endpoint"},
		{method: "GET", path: "/v1/targets/:target/directories/filterOptions"},
		{method: "POST", path: "/v1/directories:batchDelete"},
		{method: "GET", path: "/v1/targets/:target/hostPorts/filterOptions"},
		{method: "GET", path: "/v1/targets/:target/hostPorts/exportFiles/current"},
		{method: "GET", path: "/v1/targets/:target/screenshots"},
		{method: "GET", path: "/v1/targets/:target/screenshots/filterOptions"},
	}

	for _, route := range want {
		if !hasRoute(engine.Routes(), route.method, route.path) {
			t.Fatalf("missing route %s %s", route.method, route.path)
		}
	}
	for _, forbidden := range []string{
		"/v1/assets/search/",
		"/v1/assets:search/",
		"/v1/assets:exportFiles/current",
	} {
		if hasRoute(engine.Routes(), "GET", forbidden) {
			t.Fatalf("global search must not register legacy alias %s", forbidden)
		}
	}
}

func TestRegisterHealthAndSystemRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/v1")
	health := assethandler.NewHealthHandler(nil, nil)

	RegisterHealthRoutes(engine, health)
	RegisterSystemRoutes(api, health)

	want := []struct {
		method string
		path   string
	}{
		{method: "GET", path: "/healthChecks/current"},
		{method: "GET", path: "/healthChecks/liveness"},
		{method: "GET", path: "/healthChecks/readiness"},
		{method: "GET", path: "/v1/databaseHealthReports/current"},
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
