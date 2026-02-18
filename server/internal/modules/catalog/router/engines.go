package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
)

func registerEngineRoutes(protected *gin.RouterGroup, engineHandler *handler.EngineHandler) {
	protected.POST("/engines", engineHandler.Create)
	protected.GET("/engines", engineHandler.List)
	protected.GET("/engines/:id", engineHandler.GetByID)
	protected.PUT("/engines/:id", engineHandler.Update)
	protected.PATCH("/engines/:id", engineHandler.Patch)
	protected.DELETE("/engines/:id", engineHandler.Delete)
}
