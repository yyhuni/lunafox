package bootstrap

import (
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesKeepsManagementHTTPAPIs(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	registerRoutes(engine, &deps{}, func(c *gin.Context) { c.Next() })

	registered := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		registered[fmt.Sprintf("%s %s", route.Method, route.Path)] = struct{}{}
	}

	expected := []string{
		"GET /health",
		"GET /health/live",
		"GET /health/ready",
		"POST /api/auth/login",
		"POST /api/auth/refresh",
		"GET /api/auth/me",
		"POST /api/targets",
		"GET /api/targets",
		"GET /api/engines",
		"GET /api/engines/presets",
		"GET /api/wordlists",
		"GET /api/scans",
		"POST /api/scans",
		"GET /api/scans/:id/logs",
		"GET /api/vulnerabilities",
		"GET /api/system/database-health",
		"GET /api/admin/agents",
		"GET /api/admin/agents/:id",
		"GET /api/screenshots/:id/image",
	}

	for _, key := range expected {
		if _, ok := registered[key]; !ok {
			t.Fatalf("management http api missing after runtime grpc cutover: %s", key)
		}
	}
}
