package router

import (
	"github.com/gin-gonic/gin"
	assethandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler"
	directoryhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/directory"
	endpointhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/endpoint"
	hostporthandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/host_port"
	screenshothandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/screenshot"
	searchhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/search"
	subdomainhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/subdomain"
	websitehandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/website"
	snapshothandler "github.com/yyhuni/lunafox/server/internal/modules/snapshot/handler"
	snapshotrouter "github.com/yyhuni/lunafox/server/internal/modules/snapshot/router"
)

// RegisterAssetRoutes registers asset routes.
func RegisterAssetRoutes(
	api *gin.RouterGroup,
	protected *gin.RouterGroup,
	screenshotHandler *screenshothandler.ScreenshotHandler,
	screenshotSnapshotHandler *snapshothandler.ScreenshotSnapshotHandler,
	websiteHandler *websitehandler.WebsiteHandler,
	subdomainHandler *subdomainhandler.SubdomainHandler,
	endpointHandler *endpointhandler.EndpointHandler,
	directoryHandler *directoryhandler.DirectoryHandler,
	hostPortHandler *hostporthandler.HostPortHandler,
	globalAssetSearchHandler *searchhandler.GlobalAssetSearchHandler,
	assetStatisticsHandler *assethandler.AssetStatisticsHandler,
	endpointSnapshotHandler *snapshothandler.EndpointSnapshotHandler,
	hostPortSnapshotHandler *snapshothandler.HostPortSnapshotHandler,
	websiteSnapshotHandler *snapshothandler.WebsiteSnapshotHandler,
	subdomainSnapshotHandler *snapshothandler.SubdomainSnapshotHandler,
	directorySnapshotHandler *snapshothandler.DirectorySnapshotHandler,
	vulnerabilitySnapshotHandler *snapshothandler.VulnerabilitySnapshotHandler,
) {
	registerPublicRoutes(api, screenshotHandler, screenshotSnapshotHandler)
	registerWebsiteRoutes(protected, websiteHandler, websiteSnapshotHandler)
	registerSubdomainRoutes(protected, subdomainHandler)
	registerEndpointRoutes(protected, endpointHandler, endpointSnapshotHandler)
	registerDirectoryRoutes(protected, directoryHandler, directorySnapshotHandler)
	registerHostPortRoutes(protected, hostPortHandler, hostPortSnapshotHandler)
	registerGlobalAssetSearchRoutes(protected, globalAssetSearchHandler)
	registerAssetStatisticsRoutes(protected, assetStatisticsHandler)
	registerScreenshotRoutes(protected, screenshotHandler, screenshotSnapshotHandler)
	snapshotrouter.RegisterScanSnapshotRoutes(
		protected,
		websiteSnapshotHandler,
		subdomainSnapshotHandler,
		endpointSnapshotHandler,
		directorySnapshotHandler,
		hostPortSnapshotHandler,
		screenshotSnapshotHandler,
		vulnerabilitySnapshotHandler,
	)
}
