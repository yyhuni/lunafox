package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/handler"
)

// RegisterHealthRoutes registers health endpoints.
func RegisterHealthRoutes(engine *gin.Engine, healthHandler *handler.HealthHandler) {
	engine.GET("/healthChecks/current", healthHandler.Check)
	engine.GET("/healthChecks/liveness", healthHandler.Liveness)
	engine.GET("/healthChecks/readiness", healthHandler.Readiness)
}
