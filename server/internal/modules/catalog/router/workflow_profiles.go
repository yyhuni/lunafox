package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
)

// registerWorkflowProfileRoutes registers workflow profile routes.
func registerWorkflowProfileRoutes(protected *gin.RouterGroup, workflowProfileHandler *handler.WorkflowProfileHandler) {
	protected.GET("/workflows/profiles", workflowProfileHandler.List)
	protected.GET("/workflows/profiles/:id", workflowProfileHandler.GetByID)
}
