package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/system/handler"
)

func TestRegisterSystemRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	api := engine.Group("/v1")
	RegisterSystemRoutes(api, handler.NewServerLogHandler(nil), handler.NewRuntimeMetricsHandler(nil))

	want := map[string]bool{
		"GET /v1/admin/system/logEntries":             false,
		"GET /v1/admin/system/runtimeMetrics/current": false,
	}
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Fatalf("expected %s to be registered", route)
		}
	}
}
