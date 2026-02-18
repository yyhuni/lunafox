package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/auth"
)

// ContextKey for user claims
const UserClaimsKey = "userClaims"

// AuthMiddleware creates a JWT authentication middleware
func AuthMiddleware(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			return
		}

		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			status := http.StatusUnauthorized
			message := "Invalid token"

			if err == auth.ErrExpiredToken {
				message = "Token has expired"
			}

			c.AbortWithStatusJSON(status, gin.H{
				"error": message,
			})
			return
		}

		// Store claims in context
		c.Set(UserClaimsKey, claims)
		c.Next()
	}
}

// GetUserClaims retrieves user claims from context
func GetUserClaims(c *gin.Context) (*auth.Claims, bool) {
	value, exists := c.Get(UserClaimsKey)
	if !exists {
		return nil, false
	}

	claims, ok := value.(*auth.Claims)
	return claims, ok
}

// GetUserID retrieves user ID from context
func GetUserID(c *gin.Context) (int, bool) {
	claims, ok := GetUserClaims(c)
	if !ok {
		return 0, false
	}
	return claims.UserID, true
}
