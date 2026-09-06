package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/dto"
)

// AuthMiddleware creates a JWT authentication middleware.
func AuthMiddleware(jwtManager *auth.JWTManager, tokenVersions auth.TokenVersionReader) gin.HandlerFunc {
	if jwtManager == nil {
		panic("jwt manager is required")
	}
	if tokenVersions == nil {
		panic("token version reader is required")
	}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			return
		}

		tokenString := parts[1]
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			if err == auth.ErrExpiredToken {
				// Fetch-SSE reconnects distinguish an expired access token from an
				// invalid session so it can make its one allowed shared renewal.
				dto.Error(c, http.StatusUnauthorized, "TOKEN_EXPIRED", "Token has expired")
				c.Abort()
				return
			}
			status := http.StatusUnauthorized

			c.AbortWithStatusJSON(status, gin.H{
				"error": "Invalid token",
			})
			return
		}

		currentTokenVersion, err := tokenVersions.GetTokenVersion(c.Request.Context(), claims.UserID)
		if err != nil || currentTokenVersion != claims.TokenVersion {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			return
		}

		setUserClaims(c, claims)
		c.Next()
	}
}
