package router

import (
	"github.com/gin-gonic/gin"
	hostporthandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/host_port"
	snapshothandler "github.com/yyhuni/lunafox/server/internal/modules/snapshot/handler"
)

func registerHostPortRoutes(
	protected *gin.RouterGroup,
	hostPortHandler *hostporthandler.HostPortHandler,
	hostPortSnapshotHandler *snapshothandler.HostPortSnapshotHandler,
) {
	protected.GET("/targets/:id/host-ports", hostPortHandler.List)
	protected.GET("/targets/:id/host-ports/export", hostPortHandler.Export)
	protected.POST("/targets/:id/host-ports/bulk-upsert", hostPortSnapshotHandler.BulkUpsert)

	protected.POST("/host-ports/bulk-delete", hostPortHandler.BulkDelete)
}
