package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// WorkerAuthMiddleware creates a simple token authentication middleware for workers
// Workers use a static token via X-Worker-Token header (not JWT)
func WorkerAuthMiddleware(workerToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Worker-Token")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "X-Worker-Token header required",
			})
			return
		}

		if token != workerToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid worker token",
			})
			return
		}

		c.Next()
	}
}
