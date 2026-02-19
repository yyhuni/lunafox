package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// getForwardedIP returns the best-effort client IP for agent requests.
// Trust X-Forwarded-For/X-Real-IP only when requests come through trusted proxies.
func getForwardedIP(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	if realIP := strings.TrimSpace(c.GetHeader("X-Real-IP")); realIP != "" {
		return realIP
	}
	return c.ClientIP()
}
