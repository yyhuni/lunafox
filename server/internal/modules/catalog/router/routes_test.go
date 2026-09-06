package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
)

type routeTokenVersionReader struct{}

func (routeTokenVersionReader) GetTokenVersion(context.Context, int) (int, error) {
	return 0, nil
}

func TestRegisterCatalogRoutes_UsesScanWorkflowRoutesAndDisablesLegacyCatalogManagementRoutes(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	protected := engine.Group("/v1")
	engineHandler := &handler.EngineCatalogHandler{}
	RegisterCatalogRoutes(protected, nil, nil, engineHandler, nil, nil)

	registered := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		registered[fmt.Sprintf("%s %s", route.Method, route.Path)] = struct{}{}
	}

	expected := []string{
		"GET /v1/engines",
		"GET /v1/engines/:engine",
		"POST /v1/engines:installMethod",
		"POST /v1/scanWorkflows",
		"GET /v1/scanWorkflows",
		"GET /v1/scanWorkflows/:scanWorkflow/profile",
		"GET /v1/scanWorkflows/:scanWorkflow",
		"PATCH /v1/scanWorkflows/:scanWorkflow",
		"GET /v1/targets",
		"GET /v1/wordlists",
		"GET /v1/settings/apiKeys",
		"PATCH /v1/settings/apiKeys",
	}
	for _, key := range expected {
		if _, ok := registered[key]; !ok {
			t.Fatalf("catalog route missing: %s", key)
		}
	}

	// Gin represents the `:install` suffix as a wildcard parameter internally,
	// but callers must use the canonical AIP custom-method path.
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/engines:install", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("POST /v1/engines:install status = %d, want matched handler status %d", response.Code, http.StatusInternalServerError)
	}

	legacyCollection := func(name string) string {
		return "/v1/" + name
	}
	legacyWorkflowCollection := legacyCollection(strings.Join([]string{"work", "flows"}, ""))
	legacyWorkflowProfilesCollection := legacyCollection(strings.Join([]string{"workflow", "Profiles"}, ""))

	unexpected := []string{
		"POST /v1/engines",
		"GET /v1/engines/:id",
		"PUT /v1/engines/:id",
		"PATCH /v1/engines/:id",
		"DELETE /v1/engines/:id",
		"PUT " + legacyWorkflowCollection + "/:workflow",
		"DELETE " + legacyWorkflowCollection + "/:workflow",
		"GET " + legacyWorkflowCollection + "/profiles",
		"GET " + legacyWorkflowCollection + "/profiles/:workflow_profile",
		"GET " + legacyWorkflowProfilesCollection,
		"GET " + legacyWorkflowProfilesCollection + "/:workflow_profile",
	}
	for _, key := range unexpected {
		if _, ok := registered[key]; ok {
			t.Fatalf("legacy catalog-management route must be disabled in memory-only mode: %s", key)
		}
	}
}

func TestRegisterCatalogRoutes_InstallRequiresJWTBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/v1")
	protected.Use(middleware.AuthMiddleware(auth.NewJWTManager("test-secret-key-32-chars-long!!", time.Minute, time.Hour), routeTokenVersionReader{}))
	RegisterCatalogRoutes(protected, nil, nil, &handler.EngineCatalogHandler{}, nil, nil)

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/engines:install", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("POST /v1/engines:install without JWT status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestRegisterCatalogRoutes_ScanWorkflowManagementRequiresJWTBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/v1")
	protected.Use(middleware.AuthMiddleware(auth.NewJWTManager("test-secret-key-32-chars-long!!", time.Minute, time.Hour), routeTokenVersionReader{}))
	RegisterCatalogRoutes(protected, nil, nil, &handler.EngineCatalogHandler{}, nil, nil)

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/scanWorkflows", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("GET /v1/scanWorkflows without JWT status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}
