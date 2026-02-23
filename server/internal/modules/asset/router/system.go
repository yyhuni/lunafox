package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/handler"
)

// RegisterSystemRoutes registers system-facing routes under /api.
func RegisterSystemRoutes(protected *gin.RouterGroup, healthHandler *handler.HealthHandler) {
	protected.GET("/system/database-health", healthHandler.DatabaseHealth)
}
