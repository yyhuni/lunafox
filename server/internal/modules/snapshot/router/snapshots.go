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
	protected.POST("/scans/:scan/websites:batchIngest", websiteSnapshotHandler.BatchIngest)
	protected.GET("/scans/:scan/websites", websiteSnapshotHandler.List)
	protected.GET("/scans/:scan/websites/filterOptions", websiteSnapshotHandler.FilterOptions)
	protected.GET("/scans/:scan/websites/exportFiles/current", websiteSnapshotHandler.Export)

	protected.POST("/scans/:scan/subdomains:batchIngest", subdomainSnapshotHandler.BatchIngest)
	protected.GET("/scans/:scan/subdomains", subdomainSnapshotHandler.List)
	protected.GET("/scans/:scan/subdomains/exportFiles/current", subdomainSnapshotHandler.Export)

	protected.POST("/scans/:scan/endpoints:batchIngest", endpointSnapshotHandler.BatchIngest)
	protected.GET("/scans/:scan/endpoints", endpointSnapshotHandler.List)
	protected.GET("/scans/:scan/endpoints/filterOptions", endpointSnapshotHandler.FilterOptions)
	protected.GET("/scans/:scan/endpoints/exportFiles/current", endpointSnapshotHandler.Export)

	protected.POST("/scans/:scan/directories:batchIngest", directorySnapshotHandler.BatchIngest)
	protected.GET("/scans/:scan/directories", directorySnapshotHandler.List)
	protected.GET("/scans/:scan/directories/filterOptions", directorySnapshotHandler.FilterOptions)
	protected.GET("/scans/:scan/directories/exportFiles/current", directorySnapshotHandler.Export)

	protected.POST("/scans/:scan/hostPorts:batchIngest", hostPortSnapshotHandler.BatchIngest)
	protected.GET("/scans/:scan/hostPorts", hostPortSnapshotHandler.List)
	protected.GET("/scans/:scan/hostPorts/filterOptions", hostPortSnapshotHandler.FilterOptions)
	protected.GET("/scans/:scan/hostPorts/exportFiles/current", hostPortSnapshotHandler.Export)

	protected.POST("/scans/:scan/screenshots:batchIngest", screenshotSnapshotHandler.BatchIngest)
	protected.GET("/scans/:scan/screenshots", screenshotSnapshotHandler.List)
	protected.GET("/scans/:scan/screenshots/filterOptions", screenshotSnapshotHandler.FilterOptions)

	protected.POST("/scans/:scan/vulnerabilities:batchCreate", vulnerabilitySnapshotHandler.BatchCreate)
	protected.GET("/scans/:scan/vulnerabilities", vulnerabilitySnapshotHandler.ListByScan)
	protected.GET("/scans/:scan/vulnerabilities/filterOptions", vulnerabilitySnapshotHandler.FilterOptions)
	protected.GET("/scans/:scan/vulnerabilities/exportFiles/current", vulnerabilitySnapshotHandler.Export)

	protected.GET("/vulnerabilitySnapshots", vulnerabilitySnapshotHandler.ListAcrossScans)
	protected.GET("/vulnerabilitySnapshots/:vulnerability_snapshot", vulnerabilitySnapshotHandler.GetByID)
}
