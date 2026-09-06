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
	FindByAuthenticationToken(ctx context.Context, authenticationToken string) (*agentdomain.Agent, error)
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

		authenticationToken := parts[1]
		if len(authenticationToken) != 8 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token format"})
			c.Abort()
			return
		}

		agent, err := agentRepo.FindByAuthenticationToken(c.Request.Context(), authenticationToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
			c.Abort()
			return
		}

		setAgent(c, agent)
		c.Next()
	}
}
