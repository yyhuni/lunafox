package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
	snapshothandler "github.com/yyhuni/lunafox/server/internal/modules/snapshot/handler"
)

// RegisterCatalogRoutes registers target/engine/wordlist/preset routes.
func RegisterCatalogRoutes(
	api *gin.RouterGroup,
	protected *gin.RouterGroup,
	workerToken string,
	workerHandler *handler.WorkerHandler,
	wordlistHandler *handler.WordlistHandler,
	subdomainSnapshotHandler *snapshothandler.SubdomainSnapshotHandler,
	websiteSnapshotHandler *snapshothandler.WebsiteSnapshotHandler,
	endpointSnapshotHandler *snapshothandler.EndpointSnapshotHandler,
	targetHandler *handler.TargetHandler,
	engineHandler *handler.EngineHandler,
	presetHandler *handler.PresetHandler,
) {
	registerWorkerRoutes(api, workerToken, workerHandler, wordlistHandler, subdomainSnapshotHandler, websiteSnapshotHandler, endpointSnapshotHandler)
	registerTargetRoutes(protected, targetHandler)
	registerPresetRoutes(protected, presetHandler)
	registerEngineRoutes(protected, engineHandler)
	registerWordlistRoutes(protected, wordlistHandler)
}
