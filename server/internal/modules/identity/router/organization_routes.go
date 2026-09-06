package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/handler"
)

func registerOrganizationRoutes(protected *gin.RouterGroup, orgHandler *handler.OrganizationHandler) {
	protected.POST("/organizations", orgHandler.CreateOrganization)
	protected.POST("/organizations:batchDelete", orgHandler.BatchDeleteOrganizations)
	protected.GET("/organizations", orgHandler.List)
	protected.GET("/organizations/:organization", orgHandler.GetOrganizationByID)
	protected.GET("/organizations/:organization/targets", orgHandler.ListOrganizationTargets)
	// Gin cannot register sibling `:batchLink` and `:batchUnlink` routes under
	// the same static prefix, so this wildcard keeps the public AIP-style verbs.
	protected.POST("/organizations/:organization/targets:batchMethod", func(c *gin.Context) {
		httpdto.DispatchCustomMethod(c, "batchMethod", map[string]gin.HandlerFunc{
			"batchLink":   orgHandler.BatchLinkOrganizationTargets,
			"batchUnlink": orgHandler.BatchUnlinkOrganizationTargets,
		})
	})
	protected.PATCH("/organizations/:organization", orgHandler.UpdateOrganization)
	protected.DELETE("/organizations/:organization", orgHandler.DeleteOrganization)
}
