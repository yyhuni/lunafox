package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

// Recovery returns a gin middleware for recovering from panics.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := GetRequestID(c)
				stack := string(debug.Stack())

				pkg.Error("Panic recovered",
					pkg.RequestIDField(requestID),
					zap.Any("error", err),
					zap.String("stack", stack),
					zap.String("http.request.method", c.Request.Method),
					zap.String("url.path", c.Request.URL.Path),
					zap.String("client.address", c.ClientIP()),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":     "Internal Server Error",
					"requestId": requestID,
				})
			}
		}()

		c.Next()
	}
}
