package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type agentFinderStub struct {
	agent *agentdomain.Agent
	err   error
}

func (stub *agentFinderStub) FindByAPIKey(context.Context, string) (*agentdomain.Agent, error) {
	return stub.agent, stub.err
}

func TestAgentAuthMiddleware_StoresAgentIdentityViaAccessors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AgentAuthMiddleware(&agentFinderStub{agent: &agentdomain.Agent{ID: 7, Name: "agent-7"}}))
	router.GET("/protected", func(c *gin.Context) {
		agentID, ok := GetAgentID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "missing agent id"})
			return
		}
		agent, ok := GetAgent(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "missing agent"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"agentId": agentID, "name": agent.Name})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer abcd1234")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAgentAuthMiddleware_RejectsMissingBearerAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AgentAuthMiddleware(&agentFinderStub{agent: &agentdomain.Agent{ID: 7, Name: "agent-7"}}))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}
