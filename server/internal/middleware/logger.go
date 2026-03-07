package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

const (
	// RequestIDHeader is the header name for request ID.
	RequestIDHeader = "Request-Id"
)

// Logger returns a gin middleware for logging requests.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		setRequestID(c, requestID)
		c.Header(RequestIDHeader, requestID)

		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		fields := []zap.Field{
			pkg.RequestIDField(requestID),
			zap.Int("http.response.status_code", c.Writer.Status()),
			zap.String("http.request.method", c.Request.Method),
			zap.String("url.path", path),
			zap.String("url.query", query),
			zap.String("client.address", c.ClientIP()),
			zap.String("user_agent.original", c.Request.UserAgent()),
			zap.Int64("http.server.request.duration_ms", latency.Milliseconds()),
			zap.Int("http.response.body.size", c.Writer.Size()),
		}
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error", c.Errors.String()))
		}

		status := c.Writer.Status()
		switch {
		case status >= 500:
			pkg.Error("Server error", fields...)
		case status >= 400:
			pkg.Warn("Client error", fields...)
		default:
			pkg.Info("Request completed", fields...)
		}
	}
}
