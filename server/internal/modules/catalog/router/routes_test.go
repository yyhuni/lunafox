package router

import (
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterCatalogRoutes_UsesPresetRoutesAndDisablesLegacyCatalogManagementRoutes(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	protected := engine.Group("/api")
	RegisterCatalogRoutes(protected, nil, nil, nil, nil)

	registered := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		registered[fmt.Sprintf("%s %s", route.Method, route.Path)] = struct{}{}
	}

	expected := []string{
		"GET /api/workflows",
		"GET /api/workflows/:name",
		"GET /api/workflows/profiles",
		"GET /api/workflows/profiles/:id",
		"GET /api/targets",
		"GET /api/wordlists",
	}
	for _, key := range expected {
		if _, ok := registered[key]; !ok {
			t.Fatalf("catalog route missing: %s", key)
		}
	}

	unexpected := []string{
		"GET /api/engines",
		"POST /api/engines",
		"GET /api/engines/:id",
		"PUT /api/engines/:id",
		"PATCH /api/engines/:id",
		"DELETE /api/engines/:id",
		"POST /api/workflows",
		"PUT /api/workflows/:name",
		"PATCH /api/workflows/:name",
		"DELETE /api/workflows/:name",
	}
	for _, key := range unexpected {
		if _, ok := registered[key]; ok {
			t.Fatalf("legacy catalog-management route must be disabled in memory-only mode: %s", key)
		}
	}
}
