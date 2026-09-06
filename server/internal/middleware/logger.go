package middleware

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

const (
	// RequestIDHeader is the header name for request ID.
	RequestIDHeader    = "Request-Id"
	redactedQueryValue = "[REDACTED]"
)

var (
	queryKeyNormalizer       = strings.NewReplacer("-", "", "_", "", ".", "")
	sensitiveQueryKeyMarkers = [...]string{
		"token",
		"secret",
		"password",
		"authorization",
		"credential",
		"signature",
		"apikey",
	}
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
		query := sanitizeQueryForLogging(c.Request.URL.RawQuery)

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
		case status == 499:
			pkg.Info("Request cancelled", fields...)
		case status == 504:
			pkg.Warn("Request deadline exceeded", fields...)
		case status >= 500:
			pkg.Error("Server error", fields...)
		case status >= 400:
			pkg.Warn("Client error", fields...)
		default:
			pkg.Info("Request completed", fields...)
		}
	}
}

// sanitizeQueryForLogging never falls back to raw text because malformed input
// can still contain a bearer credential that must not enter request logs.
func sanitizeQueryForLogging(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}

	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return ""
	}
	for key := range query {
		if isSensitiveQueryKey(key) {
			query[key] = []string{redactedQueryValue}
		}
	}
	return query.Encode()
}

func isSensitiveQueryKey(key string) bool {
	normalized := queryKeyNormalizer.Replace(strings.ToLower(key))
	for _, marker := range sensitiveQueryKeyMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
