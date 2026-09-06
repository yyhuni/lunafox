package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/handler"
)

func TestRegisterUserRoutesIncludesMCPKeyLifecycleEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/v1")
	registerUserRoutes(protected, handler.NewUserHandler(nil))

	if !hasUserRoute(engine.Routes(), http.MethodGet, "/v1/users/me/mcpKey") {
		t.Fatal("missing MCP key status route")
	}
	if !hasUserRoute(engine.Routes(), http.MethodPost, "/v1/users/me:customMethod") {
		t.Fatal("missing user custom method dispatch route")
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/users/me:generateMcpKey", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("canonical generate action was not dispatched: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func hasUserRoute(routes gin.RoutesInfo, method, path string) bool {
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return true
		}
	}
	return false
}
