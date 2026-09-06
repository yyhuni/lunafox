package bootstrap

import (
	"github.com/gin-gonic/gin"
	agentrouter "github.com/yyhuni/lunafox/server/internal/modules/agent/router"
	assetrouter "github.com/yyhuni/lunafox/server/internal/modules/asset/router"
	blacklistrouter "github.com/yyhuni/lunafox/server/internal/modules/blacklist/router"
	catalogrouter "github.com/yyhuni/lunafox/server/internal/modules/catalog/router"
	fingerprintrouter "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/router"
	identityrouter "github.com/yyhuni/lunafox/server/internal/modules/identity/router"
	loginvisualrouter "github.com/yyhuni/lunafox/server/internal/modules/loginvisual/router"
	notificationrouter "github.com/yyhuni/lunafox/server/internal/modules/notification/router"
	nucleipocrouter "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/router"
	scanrouter "github.com/yyhuni/lunafox/server/internal/modules/scan/router"
	scheduledscanrouter "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/router"
	securityrouter "github.com/yyhuni/lunafox/server/internal/modules/security/router"
	systemrouter "github.com/yyhuni/lunafox/server/internal/modules/system/router"
)

func registerRoutes(engine *gin.Engine, d *deps, jwtMiddleware gin.HandlerFunc) {
	// MCP shares the Server lifecycle and public edge with REST, but uses its
	// own static-key admission and protocol handler instead of JWT routing.
	engine.Any("/mcp", gin.WrapH(d.mcpHandler))

	assetrouter.RegisterHealthRoutes(engine, d.healthHandler)

	api := engine.Group("/v1")
	protected := api.Group("")
	protected.Use(jwtMiddleware)

	identityrouter.RegisterIdentityRoutes(api, protected, d.authHandler, d.userHandler, d.orgHandler)
	catalogrouter.RegisterCatalogRoutes(
		protected,
		d.wordlistHandler,
		d.targetHandler,
		d.engineCatalogHandler,
		d.scanWorkflowCatalogHandler,
		d.subfinderAPIKeySettingsHandler,
	)
	blacklistrouter.RegisterBlacklistPolicyRoutes(protected, d.blacklistPolicyHandler)
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
		d.globalAssetSearchHandler,
		d.assetStatisticsHandler,
		d.endpointSnapshotHandler,
		d.hostPortSnapshotHandler,
		d.websiteSnapshotHandler,
		d.subdomainSnapshotHandler,
		d.directorySnapshotHandler,
		d.vulnerabilitySnapshotHandler,
	)
	assetrouter.RegisterSystemRoutes(protected, d.healthHandler)
	fingerprintrouter.RegisterFingerprintRoutes(protected, d.fingerprintHandler)
	scanrouter.RegisterScanRoutes(protected, d.scanHandler, d.taskProgressLogHandler)
	scheduledscanrouter.RegisterScheduledScanRoutes(protected, d.scheduledScanHandler)
	securityrouter.RegisterSecurityRoutes(protected, d.vulnerabilityHandler)
	if d.notificationInboxHandler != nil || d.notificationDestinationHandler != nil || d.notificationSSEHandler != nil {
		notificationrouter.RegisterNotificationRoutes(protected, d.notificationInboxHandler, d.notificationDestinationHandler, d.notificationSSEHandler)
	}
	if d.loginVisualHandler != nil {
		loginvisualrouter.RegisterLoginVisualRoutes(api, protected, d.loginVisualHandler)
	}
	if d.nucleiPocHandler != nil {
		nucleipocrouter.RegisterNucleiPOCRoutes(protected, d.nucleiPocHandler)
	}
	systemrouter.RegisterSystemRoutes(protected, d.serverLogHandler, d.runtimeMetricsHandler)
	agentrouter.RegisterAgentRoutes(api, protected, d.agentHandler, d.agentLogHandler, d.agentClusterSummaryHandler, d.agentLocationMapHandler)
}
