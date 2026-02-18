package router

import (
	"github.com/gin-gonic/gin"
	directoryhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/directory"
	subdomainhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/subdomain"
	websitehandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/website"
	snapshothandler "github.com/yyhuni/lunafox/server/internal/modules/snapshot/handler"
)

func registerWebsiteRoutes(
	protected *gin.RouterGroup,
	websiteHandler *websitehandler.WebsiteHandler,
	websiteSnapshotHandler *snapshothandler.WebsiteSnapshotHandler,
) {
	protected.GET("/targets/:id/websites", websiteHandler.List)
	protected.GET("/targets/:id/websites/export", websiteHandler.Export)
	protected.POST("/targets/:id/websites/bulk-create", websiteHandler.BulkCreate)
	protected.POST("/targets/:id/websites/bulk-upsert", websiteSnapshotHandler.BulkUpsert)

	protected.DELETE("/websites/:id", websiteHandler.Delete)
	protected.POST("/websites/bulk-delete", websiteHandler.BulkDelete)
}

func registerSubdomainRoutes(protected *gin.RouterGroup, subdomainHandler *subdomainhandler.SubdomainHandler) {
	protected.GET("/targets/:id/subdomains", subdomainHandler.List)
	protected.GET("/targets/:id/subdomains/export", subdomainHandler.Export)
	protected.POST("/targets/:id/subdomains/bulk-create", subdomainHandler.BulkCreate)

	protected.POST("/subdomains/bulk-delete", subdomainHandler.BulkDelete)
}

func registerDirectoryRoutes(
	protected *gin.RouterGroup,
	directoryHandler *directoryhandler.DirectoryHandler,
	directorySnapshotHandler *snapshothandler.DirectorySnapshotHandler,
) {
	protected.GET("/targets/:id/directories", directoryHandler.List)
	protected.GET("/targets/:id/directories/export", directoryHandler.Export)
	protected.POST("/targets/:id/directories/bulk-create", directoryHandler.BulkCreate)
	protected.POST("/targets/:id/directories/bulk-upsert", directorySnapshotHandler.BulkUpsert)

	protected.POST("/directories/bulk-delete", directoryHandler.BulkDelete)
}
