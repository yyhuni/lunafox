package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/handler"
)

// RegisterScanRoutes registers scan and scan-log routes.
func RegisterScanRoutes(
	protected *gin.RouterGroup,
	scanHandler *handler.ScanHandler,
	scanLogHandler *handler.ScanLogHandler,
) {
	registerScanRoutes(protected, scanHandler)
	registerScanLogRoutes(protected, scanLogHandler)
}
