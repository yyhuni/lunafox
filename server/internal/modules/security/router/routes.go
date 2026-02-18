package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/security/handler"
)

// RegisterSecurityRoutes registers security module routes.
func RegisterSecurityRoutes(protected *gin.RouterGroup, vulnerabilityHandler *handler.VulnerabilityHandler) {
	registerVulnerabilityRoutes(protected, vulnerabilityHandler)
}
