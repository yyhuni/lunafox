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
	protected.GET("/targets/:target/hostPorts", hostPortHandler.List)
	protected.GET("/targets/:target/hostPorts/filterOptions", hostPortHandler.FilterOptions)
	protected.GET("/targets/:target/hostPorts/exportFiles/current", hostPortHandler.Export)
	protected.POST("/targets/:target/hostPorts:batchIngest", hostPortSnapshotHandler.BatchIngest)

	protected.POST("/hostPorts:batchDelete", hostPortHandler.BatchDelete)
}
