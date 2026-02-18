package router

import (
	"github.com/gin-gonic/gin"
	screenshothandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/screenshot"
	snapshothandler "github.com/yyhuni/lunafox/server/internal/modules/snapshot/handler"
)

func registerScreenshotRoutes(
	protected *gin.RouterGroup,
	screenshotHandler *screenshothandler.ScreenshotHandler,
	screenshotSnapshotHandler *snapshothandler.ScreenshotSnapshotHandler,
) {
	protected.GET("/targets/:id/screenshots", screenshotHandler.ListByTargetID)
	protected.POST("/targets/:id/screenshots/bulk-upsert", screenshotSnapshotHandler.BulkUpsert)

	protected.POST("/screenshots/bulk-delete", screenshotHandler.BulkDelete)
}
