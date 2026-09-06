package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
)

func registerScanWorkflowCatalogRoutes(protected *gin.RouterGroup, scanWorkflowHandler *handler.ScanWorkflowManagementHandler) {
	protected.POST("/scanWorkflows", scanWorkflowHandler.Create)
	protected.GET("/scanWorkflows", scanWorkflowHandler.List)
	protected.GET("/scanWorkflows/:scanWorkflow/profile", scanWorkflowHandler.GetProfile)
	protected.GET("/scanWorkflows/:scanWorkflow", scanWorkflowHandler.Get)
	protected.PATCH("/scanWorkflows/:scanWorkflow", scanWorkflowHandler.Update)
}
