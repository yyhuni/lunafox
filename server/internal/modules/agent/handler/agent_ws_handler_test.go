package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentinfra "github.com/yyhuni/lunafox/server/internal/modules/agent/infrastructure"
	ws "github.com/yyhuni/lunafox/server/internal/websocket"
)

func TestAgentWSHandlerUnauthorizedWithoutAgentContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAgentWebSocketHandler(ws.NewHub(), agentapp.NewAgentRuntimeService(nil, nil, nil, agentinfra.NewSystemClock(), "", ""))
	router := gin.New()
	router.GET("/api/agent/ws", handler.Handle)

	req := httptest.NewRequest(http.MethodGet, "/api/agent/ws", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
