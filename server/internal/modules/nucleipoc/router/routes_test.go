package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/handler"
)

func TestRegisterNucleiPOCRoutesExposesOnlyReplacementBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/v1")
	RegisterNucleiPOCRoutes(protected, handler.NewNucleiPOCHandler(nil))
	routes := engine.Routes()
	seen := make(map[string]string, len(routes))
	for _, route := range routes {
		seen[route.Method+" "+route.Path] = route.Handler
	}
	for _, expected := range []string{
		http.MethodPost + " /v1/nucleiPocSources:sync",
		http.MethodGet + " /v1/nucleiPocSources/current",
		http.MethodGet + " /v1/nucleiPocSyncTasks/:task",
		http.MethodGet + " /v1/nucleiPocs",
		http.MethodGet + " /v1/nucleiPocs/filterOptions",
		http.MethodPost + " /v1/nucleiPocs:setActivation",
		http.MethodGet + " /v1/nucleiPocs/:nucleiPoc",
		http.MethodPatch + " /v1/nucleiPocs/:nucleiPoc",
	} {
		if _, ok := seen[expected]; !ok {
			t.Fatalf("missing route %s", expected)
		}
	}
	for route := range seen {
		if len(route) >= len("/nuclei/repos") && route[len(route)-len("/nuclei/repos"):] == "/nuclei/repos" {
			t.Fatalf("legacy route registered: %s", route)
		}
	}
}

func TestRegisterNucleiPOCRoutesKeepsCollectionActivationUnderProtectedGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/v1")
	protected.Use(func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	})
	RegisterNucleiPOCRoutes(protected, handler.NewNucleiPOCHandler(nil))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/nucleiPocs:setActivation", strings.NewReader(`{"enabled":true}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated collection activation status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
