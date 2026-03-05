package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
)

func registerWorkflowRoutes(protected *gin.RouterGroup, workflowHandler *handler.WorkflowHandler) {
	protected.GET("/workflows", workflowHandler.List)
	protected.GET("/workflows/:workflowId", workflowHandler.GetByWorkflowID)
}
