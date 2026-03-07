package middleware

import (
	"context"
	"net/http"
	"strings"

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
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		apiKey := parts[1]
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

		setAgent(c, agent)
		c.Next()
	}
}
