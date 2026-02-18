package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
)

func TestAgentTaskHandlerRejectsInvalidStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := agentapp.NewAgentTaskService(nil)
	handler := NewAgentTaskHandler(service)

	router := gin.New()
	router.PATCH("/api/agent/tasks/:taskId/status", func(c *gin.Context) {
		c.Set("agentID", 1)
		handler.UpdateTaskStatus(c)
	})

	body, _ := json.Marshal(map[string]string{"status": "running"})
	req := httptest.NewRequest(http.MethodPatch, "/api/agent/tasks/1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
