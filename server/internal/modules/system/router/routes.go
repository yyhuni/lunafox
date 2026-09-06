package router

import (
	"github.com/gin-gonic/gin"
	systemhandler "github.com/yyhuni/lunafox/server/internal/modules/system/handler"
)

func RegisterSystemRoutes(
	protected *gin.RouterGroup,
	serverLogHandler *systemhandler.ServerLogHandler,
	runtimeMetricsHandler *systemhandler.RuntimeMetricsHandler,
) {
	admin := protected.Group("/admin/system")
	{
		admin.GET("/logEntries", serverLogHandler.List)
		admin.GET("/runtimeMetrics/current", runtimeMetricsHandler.Current)
	}
}
