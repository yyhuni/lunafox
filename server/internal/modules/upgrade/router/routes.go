package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/handler"
)

// RegisterUpgradeRoutes registers only the canonical versioned upgrade
// boundaries. All routes are expected to be mounted under the JWT-protected
// `/v1` group by bootstrap.
func RegisterUpgradeRoutes(protected *gin.RouterGroup, upgradeHandler *handler.UpgradeHandler) {
	if protected == nil || upgradeHandler == nil {
		panic("upgrade route dependencies are required")
	}
	protected.POST("/system:checkForUpdates", upgradeHandler.CheckForUpdates)
	protected.POST("/upgradeOperations", upgradeHandler.CreateOperation)
	protected.GET("/upgradeOperations/:upgradeOperation", upgradeHandler.GetOperation)
	protected.POST("/upgradeOperations/*upgradeOperationAction", upgradeHandler.RetryAction)
}
