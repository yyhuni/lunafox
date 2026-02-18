package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	cataloghandler "github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
	snapshothandler "github.com/yyhuni/lunafox/server/internal/modules/snapshot/handler"
)

func registerWorkerRoutes(
	api *gin.RouterGroup,
	workerToken string,
	workerHandler *cataloghandler.WorkerHandler,
	wordlistHandler *cataloghandler.WordlistHandler,
	subdomainSnapshotHandler *snapshothandler.SubdomainSnapshotHandler,
	websiteSnapshotHandler *snapshothandler.WebsiteSnapshotHandler,
	endpointSnapshotHandler *snapshothandler.EndpointSnapshotHandler,
) {
	workerAPI := api.Group("/worker")
	workerAPI.Use(middleware.WorkerAuthMiddleware(workerToken))
	{
		workerAPI.GET("/scans/:id/provider-config", workerHandler.GetProviderConfig)
		workerAPI.GET("/wordlists/:name", wordlistHandler.GetByName)
		workerAPI.GET("/wordlists/:name/download", wordlistHandler.DownloadByName)
		workerAPI.POST("/scans/:id/subdomains/bulk-upsert", subdomainSnapshotHandler.BulkUpsert)
		workerAPI.POST("/scans/:id/websites/bulk-upsert", websiteSnapshotHandler.BulkUpsert)
		workerAPI.POST("/scans/:id/endpoints/bulk-upsert", endpointSnapshotHandler.BulkUpsert)
	}
}
