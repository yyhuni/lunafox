package router

import (
	"github.com/gin-gonic/gin"
	endpointhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/endpoint"
	snapshothandler "github.com/yyhuni/lunafox/server/internal/modules/snapshot/handler"
)

func registerEndpointRoutes(
	protected *gin.RouterGroup,
	endpointHandler *endpointhandler.EndpointHandler,
	endpointSnapshotHandler *snapshothandler.EndpointSnapshotHandler,
) {
	protected.GET("/targets/:id/endpoints", endpointHandler.List)
	protected.GET("/targets/:id/endpoints/export", endpointHandler.Export)
	protected.POST("/targets/:id/endpoints/bulk-create", endpointHandler.BulkCreate)
	protected.POST("/targets/:id/endpoints/bulk-upsert", endpointSnapshotHandler.BulkUpsert)

	protected.GET("/endpoints/:id", endpointHandler.GetByID)
	protected.DELETE("/endpoints/:id", endpointHandler.Delete)
	protected.POST("/endpoints/bulk-delete", endpointHandler.BulkDelete)
}
