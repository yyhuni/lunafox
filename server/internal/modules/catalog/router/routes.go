package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
)

// RegisterCatalogRoutes registers target, wordlist, scan workflow catalog, and scan workflow profile catalog routes.
func RegisterCatalogRoutes(
	protected *gin.RouterGroup,
	wordlistHandler *handler.WordlistHandler,
	targetHandler *handler.TargetHandler,
	engineCatalogHandler *handler.EngineCatalogHandler,
	scanWorkflowHandler *handler.ScanWorkflowManagementHandler,
	subfinderAPIKeySettingsHandler *handler.SubfinderAPIKeySettingsHandler,
) {
	registerTargetRoutes(protected, targetHandler)
	registerEngineCatalogRoutes(protected, engineCatalogHandler)
	registerScanWorkflowCatalogRoutes(protected, scanWorkflowHandler)
	registerWordlistRoutes(protected, wordlistHandler)
	registerSubfinderAPIKeySettingsRoutes(protected, subfinderAPIKeySettingsHandler)
}
