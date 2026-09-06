package router

import (
	"github.com/gin-gonic/gin"
	endpointhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/endpoint"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	snapshothandler "github.com/yyhuni/lunafox/server/internal/modules/snapshot/handler"
)

func registerEndpointRoutes(
	protected *gin.RouterGroup,
	endpointHandler *endpointhandler.EndpointHandler,
	endpointSnapshotHandler *snapshothandler.EndpointSnapshotHandler,
) {
	protected.GET("/targets/:target/endpoints", endpointHandler.List)
	protected.GET("/targets/:target/endpoints/filterOptions", endpointHandler.FilterOptions)
	protected.GET("/targets/:target/endpoints/exportFiles/current", endpointHandler.Export)
	protected.POST("/targets/:target/endpoints:batchMethod", func(c *gin.Context) {
		httpdto.DispatchCustomMethod(c, "batchMethod", map[string]gin.HandlerFunc{
			"batchCreate": endpointHandler.BatchCreate,
			"batchIngest": endpointSnapshotHandler.BatchIngest,
		})
	})

	protected.GET("/endpoints/:endpoint", endpointHandler.GetByID)
	protected.DELETE("/endpoints/:endpoint", endpointHandler.Delete)
	protected.POST("/endpoints:batchDelete", endpointHandler.BatchDelete)
}
