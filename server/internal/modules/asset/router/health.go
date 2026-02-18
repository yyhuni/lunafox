package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/handler"
)

// RegisterHealthRoutes registers health endpoints.
func RegisterHealthRoutes(engine *gin.Engine, healthHandler *handler.HealthHandler) {
	engine.GET("/health", healthHandler.Check)
	engine.GET("/health/live", healthHandler.Liveness)
	engine.GET("/health/ready", healthHandler.Readiness)
}
