package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

// AgentFinder defines behavior required for agent auth.
type AgentFinder interface {
	FindByAPIKey(ctx context.Context, apiKey string) (*agentdomain.Agent, error)
}

// AgentAuthMiddleware creates a middleware for agent authentication.
func AgentAuthMiddleware(agentRepo AgentFinder) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-Agent-Key")
		if apiKey == "" {
			apiKey = c.Query("key")
		}
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing API key"})
			c.Abort()
			return
		}
		if len(apiKey) != 8 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key format"})
			c.Abort()
			return
		}

		agent, err := agentRepo.FindByAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}

		c.Set("agentID", agent.ID)
		c.Set("agent", agent)
		c.Next()
	}
}
