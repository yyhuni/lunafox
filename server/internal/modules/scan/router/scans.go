package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/handler"
)

func registerScanRoutes(protected *gin.RouterGroup, scanHandler *handler.ScanHandler) {
	protected.GET("/scans", scanHandler.List)
	protected.POST("/scans/normal", scanHandler.CreateNormal)
	protected.POST("/scans/quick", scanHandler.CreateQuick)
	protected.GET("/scans/stats", scanHandler.Statistics)
	protected.GET("/scans/:id", scanHandler.GetByID)
	protected.DELETE("/scans/:id", scanHandler.Delete)
	protected.DELETE("/scans/:id/permanent", scanHandler.HardDelete)
	protected.POST("/scans/:id/stoppages", scanHandler.Stop)
	protected.POST("/scans/deletions", scanHandler.BulkDelete)
}
