package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/security/handler"
)

func registerVulnerabilityRoutes(protected *gin.RouterGroup, vulnerabilityHandler *handler.VulnerabilityHandler) {
	protected.GET("/vulnerabilities", vulnerabilityHandler.ListAll)
	protected.GET("/vulnerabilities/stats", vulnerabilityHandler.GetStats)
	protected.GET("/vulnerabilities/:id", vulnerabilityHandler.GetByID)

	protected.GET("/targets/:id/vulnerabilities", vulnerabilityHandler.ListByTarget)
	protected.GET("/targets/:id/vulnerabilities/stats", vulnerabilityHandler.GetStatsByTarget)
	protected.POST("/targets/:id/vulnerabilities/bulk-create", vulnerabilityHandler.BulkCreate)

	protected.POST("/vulnerabilities/bulk-delete", vulnerabilityHandler.BulkDelete)
	protected.PATCH("/vulnerabilities/:id/review", vulnerabilityHandler.MarkAsReviewed)
	protected.PATCH("/vulnerabilities/:id/unreview", vulnerabilityHandler.MarkAsUnreviewed)
	protected.POST("/vulnerabilities/bulk-review", vulnerabilityHandler.BulkMarkAsReviewed)
	protected.POST("/vulnerabilities/bulk-unreview", vulnerabilityHandler.BulkMarkAsUnreviewed)
}
