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
	protected.GET("/targets/:target/screenshots", screenshotHandler.List)
	protected.GET("/targets/:target/screenshots/filterOptions", screenshotHandler.FilterOptions)
	protected.POST("/targets/:target/screenshots:batchIngest", screenshotSnapshotHandler.BatchIngest)

	protected.POST("/screenshots:batchDelete", screenshotHandler.BatchDelete)
}
