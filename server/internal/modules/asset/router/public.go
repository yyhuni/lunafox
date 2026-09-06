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
	api.GET("/screenshots/:screenshot/blob", screenshotHandler.GetImage)
	api.GET("/scans/:scan/screenshotSnapshots/:screenshot_snapshot/blob", screenshotSnapshotHandler.GetImage)
}
