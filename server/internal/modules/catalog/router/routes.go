package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
)

// RegisterCatalogRoutes registers target, wordlist, workflow, and workflow profile routes.
func RegisterCatalogRoutes(
	protected *gin.RouterGroup,
	wordlistHandler *handler.WordlistHandler,
	targetHandler *handler.TargetHandler,
	workflowHandler *handler.WorkflowHandler,
	workflowProfileHandler *handler.WorkflowProfileHandler,
) {
	registerTargetRoutes(protected, targetHandler)
	registerWorkflowProfileRoutes(protected, workflowProfileHandler)
	registerWorkflowRoutes(protected, workflowHandler)
	registerWordlistRoutes(protected, wordlistHandler)
}
