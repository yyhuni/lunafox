package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/handler"
)

func registerOrganizationRoutes(protected *gin.RouterGroup, orgHandler *handler.OrganizationHandler) {
	protected.POST("/organizations", orgHandler.CreateOrganization)
	protected.POST("/organizations/bulk-delete", orgHandler.BulkDeleteOrganizations)
	protected.GET("/organizations", orgHandler.ListOrganizations)
	protected.GET("/organizations/:id", orgHandler.GetOrganizationByID)
	protected.GET("/organizations/:id/targets", orgHandler.ListOrganizationTargets)
	protected.POST("/organizations/:id/link_targets", orgHandler.LinkOrganizationTargets)
	protected.POST("/organizations/:id/unlink_targets", orgHandler.UnlinkOrganizationTargets)
	protected.PUT("/organizations/:id", orgHandler.UpdateOrganization)
	protected.DELETE("/organizations/:id", orgHandler.DeleteOrganization)
}
