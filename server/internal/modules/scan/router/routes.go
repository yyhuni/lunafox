package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/handler"
)

// RegisterScanRoutes registers scan and task-progress-log routes.
func RegisterScanRoutes(
	protected *gin.RouterGroup,
	scanHandler *handler.ScanHandler,
	taskProgressLogHandler *handler.TaskProgressLogHandler,
) {
	registerScanRoutes(protected, scanHandler)
	registerTaskProgressLogRoutes(protected, taskProgressLogHandler)
}
