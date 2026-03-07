package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/auth"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

const (
	requestIDContextKey  = "requestId"
	userClaimsContextKey = "userClaims"
	agentIDContextKey    = "agentId"
	agentContextKey      = "agent"
)

func setRequestID(c *gin.Context, requestID string) {
	c.Set(requestIDContextKey, requestID)
}

// GetRequestID returns the request ID from context.
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(requestIDContextKey); exists {
		if value, ok := requestID.(string); ok {
			return value
		}
	}
	return ""
}

func setUserClaims(c *gin.Context, claims *auth.Claims) {
	c.Set(userClaimsContextKey, claims)
}

// GetUserClaims retrieves user claims from context.
func GetUserClaims(c *gin.Context) (*auth.Claims, bool) {
	value, exists := c.Get(userClaimsContextKey)
	if !exists {
		return nil, false
	}

	claims, ok := value.(*auth.Claims)
	return claims, ok
}

// GetUserID retrieves user ID from context.
func GetUserID(c *gin.Context) (int, bool) {
	claims, ok := GetUserClaims(c)
	if !ok {
		return 0, false
	}
	return claims.UserID, true
}

func setAgent(c *gin.Context, agent *agentdomain.Agent) {
	if agent == nil {
		return
	}
	c.Set(agentIDContextKey, agent.ID)
	c.Set(agentContextKey, agent)
}

// GetAgentID retrieves agent ID from context.
func GetAgentID(c *gin.Context) (int, bool) {
	value, exists := c.Get(agentIDContextKey)
	if !exists {
		return 0, false
	}
	agentID, ok := value.(int)
	return agentID, ok
}

// GetAgent retrieves the authenticated agent from context.
func GetAgent(c *gin.Context) (*agentdomain.Agent, bool) {
	value, exists := c.Get(agentContextKey)
	if !exists {
		return nil, false
	}
	agent, ok := value.(*agentdomain.Agent)
	return agent, ok
}
