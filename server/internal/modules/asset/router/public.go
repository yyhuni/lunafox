package router

import (
	"github.com/gin-gonic/gin"
	screenshothandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/screenshot"
	snapshothandler "github.com/yyhuni/lunafox/server/internal/modules/snapshot/handler"
)

func registerPublicRoutes(
	api *gin.RouterGroup,
	screenshotHandler *screenshothandler.ScreenshotHandler,
	screenshotSnapshotHandler *snapshothandler.ScreenshotSnapshotHandler,
) {
	api.GET("/screenshots/:id/image", screenshotHandler.GetImage)
	api.GET("/scans/:id/screenshots/:snapshotId/image", screenshotSnapshotHandler.GetImage)
}
