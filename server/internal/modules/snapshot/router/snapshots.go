package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/snapshot/handler"
)

func RegisterScanSnapshotRoutes(
	protected *gin.RouterGroup,
	websiteSnapshotHandler *handler.WebsiteSnapshotHandler,
	subdomainSnapshotHandler *handler.SubdomainSnapshotHandler,
	endpointSnapshotHandler *handler.EndpointSnapshotHandler,
	directorySnapshotHandler *handler.DirectorySnapshotHandler,
	hostPortSnapshotHandler *handler.HostPortSnapshotHandler,
	screenshotSnapshotHandler *handler.ScreenshotSnapshotHandler,
	vulnerabilitySnapshotHandler *handler.VulnerabilitySnapshotHandler,
) {
	protected.POST("/scans/:id/websites/bulk-upsert", websiteSnapshotHandler.BulkUpsert)
	protected.GET("/scans/:id/websites", websiteSnapshotHandler.List)
	protected.GET("/scans/:id/websites/export", websiteSnapshotHandler.Export)

	protected.POST("/scans/:id/subdomains/bulk-upsert", subdomainSnapshotHandler.BulkUpsert)
	protected.GET("/scans/:id/subdomains", subdomainSnapshotHandler.List)
	protected.GET("/scans/:id/subdomains/export", subdomainSnapshotHandler.Export)

	protected.POST("/scans/:id/endpoints/bulk-upsert", endpointSnapshotHandler.BulkUpsert)
	protected.GET("/scans/:id/endpoints", endpointSnapshotHandler.List)
	protected.GET("/scans/:id/endpoints/export", endpointSnapshotHandler.Export)

	protected.POST("/scans/:id/directories/bulk-upsert", directorySnapshotHandler.BulkUpsert)
	protected.GET("/scans/:id/directories", directorySnapshotHandler.List)
	protected.GET("/scans/:id/directories/export", directorySnapshotHandler.Export)

	protected.POST("/scans/:id/host-ports/bulk-upsert", hostPortSnapshotHandler.BulkUpsert)
	protected.GET("/scans/:id/host-ports", hostPortSnapshotHandler.List)
	protected.GET("/scans/:id/host-ports/export", hostPortSnapshotHandler.Export)

	protected.POST("/scans/:id/screenshots/bulk-upsert", screenshotSnapshotHandler.BulkUpsert)
	protected.GET("/scans/:id/screenshots", screenshotSnapshotHandler.List)

	protected.POST("/scans/:id/vulnerabilities/bulk-create", vulnerabilitySnapshotHandler.BulkCreate)
	protected.GET("/scans/:id/vulnerabilities", vulnerabilitySnapshotHandler.ListByScan)
	protected.GET("/scans/:id/vulnerabilities/export", vulnerabilitySnapshotHandler.Export)

	protected.GET("/vulnerability-snapshots", vulnerabilitySnapshotHandler.ListAll)
	protected.GET("/vulnerability-snapshots/:id", vulnerabilitySnapshotHandler.GetByID)
}
