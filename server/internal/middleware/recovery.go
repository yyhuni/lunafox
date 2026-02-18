package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

// Recovery returns a gin middleware for recovering from panics
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get request ID for tracing
				requestID := GetRequestID(c)

				// Get stack trace
				stack := string(debug.Stack())

				// Log the panic with full stack trace
				pkg.Error("Panic recovered",
					zap.String("requestId", requestID),
					zap.Any("error", err),
					zap.String("stack", stack),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("ip", c.ClientIP()),
				)

				// Return 500 error
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":     "Internal Server Error",
					"requestId": requestID,
				})
			}
		}()

		c.Next()
	}
}
