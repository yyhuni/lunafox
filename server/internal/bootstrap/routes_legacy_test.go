package bootstrap

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesExcludesLegacyRuntimeEndpoints(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	registerRoutes(engine, &deps{}, func(c *gin.Context) { c.Next() })

	legacy := map[string]struct{}{
		"GET /api/agent/ws":                                 {},
		"POST /api/agent/tasks/pull":                        {},
		"PATCH /api/agent/tasks/:taskId/status":             {},
		"GET /api/worker/scans/:id/provider-config":         {},
		"GET /api/worker/wordlists/:name":                   {},
		"GET /api/worker/wordlists/:name/download":          {},
		"POST /api/worker/scans/:id/subdomains/bulk-upsert": {},
		"POST /api/worker/scans/:id/websites/bulk-upsert":   {},
		"POST /api/worker/scans/:id/endpoints/bulk-upsert":  {},
		"GET /api/worker/scans/:id/target":                  {},
	}

	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, exists := legacy[key]; exists {
			t.Fatalf("legacy runtime route still registered: %s", key)
		}
	}
}
