package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/handler"
)

func RegisterWorkerScanRoutes(api *gin.RouterGroup, workerToken string, workerScanHandler *handler.WorkerScanHandler) {
	workerAPI := api.Group("/worker")
	workerAPI.Use(middleware.WorkerAuthMiddleware(workerToken))
	{
		workerAPI.GET("/scans/:id/target", workerScanHandler.GetTargetName)
	}
}
