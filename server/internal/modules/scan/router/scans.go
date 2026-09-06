package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/handler"
)

func registerScanRoutes(protected *gin.RouterGroup, scanHandler *handler.ScanHandler) {
	protected.GET("/scans", scanHandler.List)
	protected.POST("/scans:customMethod", func(c *gin.Context) {
		httpdto.DispatchCustomMethod(c, "customMethod", map[string]gin.HandlerFunc{
			"batchCreate": scanHandler.BatchCreate,
			"batchDelete": scanHandler.BatchDelete,
			"batchStop":   scanHandler.BatchStop,
			"quickCreate": scanHandler.CreateQuick,
		})
	})
	protected.GET("/scanStatistics", scanHandler.Statistics)
	protected.GET("/scans/:scan", scanHandler.GetByID)
	protected.POST("/scans/:scan", scanHandler.Stop)
	protected.DELETE("/scans/:scan", scanHandler.Delete)
}
