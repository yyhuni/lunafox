package router

import (
	"github.com/gin-gonic/gin"
	directoryhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/directory"
	subdomainhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/subdomain"
	websitehandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/website"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	snapshothandler "github.com/yyhuni/lunafox/server/internal/modules/snapshot/handler"
)

func registerWebsiteRoutes(
	protected *gin.RouterGroup,
	websiteHandler *websitehandler.WebsiteHandler,
	websiteSnapshotHandler *snapshothandler.WebsiteSnapshotHandler,
) {
	protected.GET("/targets/:target/websites", websiteHandler.List)
	protected.GET("/websites/:website", websiteHandler.Get)
	protected.GET("/targets/:target/websites/filterOptions", websiteHandler.FilterOptions)
	protected.GET("/targets/:target/websites/exportFiles/current", websiteHandler.Export)
	protected.POST("/targets/:target/websites:batchMethod", func(c *gin.Context) {
		httpdto.DispatchCustomMethod(c, "batchMethod", map[string]gin.HandlerFunc{
			"batchCreate": websiteHandler.BatchCreate,
			"batchIngest": websiteSnapshotHandler.BatchIngest,
		})
	})

	protected.DELETE("/websites/:website", websiteHandler.Delete)
	protected.POST("/websites:batchDelete", websiteHandler.BatchDelete)
}

func registerSubdomainRoutes(protected *gin.RouterGroup, subdomainHandler *subdomainhandler.SubdomainHandler) {
	protected.GET("/targets/:target/subdomains", subdomainHandler.List)
	protected.GET("/targets/:target/subdomains/exportFiles/current", subdomainHandler.Export)
	protected.POST("/targets/:target/subdomains:batchCreate", subdomainHandler.BatchCreate)

	protected.POST("/subdomains:batchDelete", subdomainHandler.BatchDelete)
}

func registerDirectoryRoutes(
	protected *gin.RouterGroup,
	directoryHandler *directoryhandler.DirectoryHandler,
	directorySnapshotHandler *snapshothandler.DirectorySnapshotHandler,
) {
	protected.GET("/targets/:target/directories", directoryHandler.List)
	protected.GET("/targets/:target/directories/filterOptions", directoryHandler.FilterOptions)
	protected.GET("/targets/:target/directories/exportFiles/current", directoryHandler.Export)
	protected.POST("/targets/:target/directories:batchMethod", func(c *gin.Context) {
		httpdto.DispatchCustomMethod(c, "batchMethod", map[string]gin.HandlerFunc{
			"batchCreate": directoryHandler.BatchCreate,
			"batchIngest": directorySnapshotHandler.BatchIngest,
		})
	})

	protected.POST("/directories:batchDelete", directoryHandler.BatchDelete)
}
