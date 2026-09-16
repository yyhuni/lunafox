package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/handler"
)

func RegisterScheduledScanRoutes(protected *gin.RouterGroup, scheduledScanHandler *handler.ScheduledScanHandler) {
	protected.GET("/scheduledScans", scheduledScanHandler.List)
	protected.GET("/scheduledScans:summarize", scheduledScanHandler.Summarize)
	protected.POST("/scheduledScans", scheduledScanHandler.Create)
	protected.POST("/scheduledScans:batchUpdate", scheduledScanHandler.BatchUpdate)
	protected.GET("/scheduledScans/:scheduledScan", scheduledScanHandler.GetByID)
	protected.PATCH("/scheduledScans/:scheduledScan", scheduledScanHandler.Update)
	protected.DELETE("/scheduledScans/:scheduledScan", scheduledScanHandler.Delete)
}
