package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/modules/security/handler"
)

func registerVulnerabilityRoutes(protected *gin.RouterGroup, vulnerabilityHandler *handler.VulnerabilityHandler) {
	protected.GET("/vulnerabilities", vulnerabilityHandler.List)
	protected.GET("/vulnerabilities/filterOptions", vulnerabilityHandler.FilterOptions)
	protected.GET("/vulnerabilityStatistics", vulnerabilityHandler.GetGlobalVulnerabilityStatistics)
	protected.GET("/vulnerabilities/:vulnerability", vulnerabilityHandler.GetByID)
	protected.PATCH("/vulnerabilities/:vulnerability", vulnerabilityHandler.UpdateReviewState)

	protected.GET("/targets/:target/vulnerabilities", vulnerabilityHandler.ListByTarget)
	protected.GET("/targets/:target/vulnerabilities/filterOptions", vulnerabilityHandler.FilterOptionsByTarget)
	protected.GET("/targets/:target/vulnerabilityStatistics", vulnerabilityHandler.GetTargetVulnerabilityStatistics)
	protected.POST("/targets/:target/vulnerabilities:batchCreate", vulnerabilityHandler.BatchCreate)

	protected.POST("/vulnerabilities:batchMethod", func(c *gin.Context) {
		httpdto.DispatchCustomMethod(c, "batchMethod", map[string]gin.HandlerFunc{
			"batchDelete": vulnerabilityHandler.BatchDelete,
			"batchUpdate": vulnerabilityHandler.BatchUpdate,
		})
	})
}
