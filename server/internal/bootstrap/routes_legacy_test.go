package bootstrap

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesExcludesLegacyRuntimeEndpoints(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	registerRoutes(engine, &deps{}, func(c *gin.Context) { c.Next() })

	legacyExact := map[string]struct{}{
		"GET /v1/agent/ws":                     {},
		"POST /v1/agent/tasks/pull":            {},
		"PATCH /v1/agent/tasks/:taskId/status": {},
	}

	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, exists := legacyExact[key]; exists || strings.HasPrefix(route.Path, "/v1/worker/") {
			t.Fatalf("legacy runtime route still registered: %s", key)
		}
	}
}
