package bootstrap

import (
	"github.com/gin-gonic/gin"
	agentrouter "github.com/yyhuni/lunafox/server/internal/modules/agent/router"
	assetrouter "github.com/yyhuni/lunafox/server/internal/modules/asset/router"
	catalogrouter "github.com/yyhuni/lunafox/server/internal/modules/catalog/router"
	identityrouter "github.com/yyhuni/lunafox/server/internal/modules/identity/router"
	scanrouter "github.com/yyhuni/lunafox/server/internal/modules/scan/router"
	securityrouter "github.com/yyhuni/lunafox/server/internal/modules/security/router"
)

func registerRoutes(engine *gin.Engine, d *deps, workerToken string, jwtMiddleware gin.HandlerFunc) {
	assetrouter.RegisterHealthRoutes(engine, d.healthHandler)

	api := engine.Group("/api")
	protected := api.Group("")
	protected.Use(jwtMiddleware)

	identityrouter.RegisterIdentityRoutes(api, protected, d.authHandler, d.userHandler, d.orgHandler)
	catalogrouter.RegisterCatalogRoutes(
		api,
		protected,
		workerToken,
		d.workerHandler,
		d.wordlistHandler,
		d.subdomainSnapshotHandler,
		d.websiteSnapshotHandler,
		d.endpointSnapshotHandler,
		d.targetHandler,
		d.engineHandler,
		d.presetHandler,
	)
	assetrouter.RegisterAssetRoutes(
		api,
		protected,
		d.screenshotHandler,
		d.screenshotSnapshotHandler,
		d.websiteHandler,
		d.subdomainHandler,
		d.endpointHandler,
		d.directoryHandler,
		d.hostPortHandler,
		d.endpointSnapshotHandler,
		d.hostPortSnapshotHandler,
		d.websiteSnapshotHandler,
		d.subdomainSnapshotHandler,
		d.directorySnapshotHandler,
		d.vulnerabilitySnapshotHandler,
	)
	scanrouter.RegisterScanRoutes(protected, d.scanHandler, d.scanLogHandler)
	scanrouter.RegisterWorkerScanRoutes(api, workerToken, d.workerScanHandler)
	securityrouter.RegisterSecurityRoutes(protected, d.vulnerabilityHandler)
	agentrouter.RegisterAgentRoutes(api, protected, d.agentHandler, d.agentWSHandler, d.agentTaskHandler, d.agentLogHandler, d.agentRepo)
}
