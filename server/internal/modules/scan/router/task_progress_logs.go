package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/handler"
)

func registerTaskProgressLogRoutes(protected *gin.RouterGroup, taskProgressLogHandler *handler.TaskProgressLogHandler) {
	protected.GET("/scans/:scan/taskProgressLogs", taskProgressLogHandler.List)
}
