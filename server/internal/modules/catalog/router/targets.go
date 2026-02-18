package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
)

func registerTargetRoutes(protected *gin.RouterGroup, targetHandler *handler.TargetHandler) {
	protected.POST("/targets", targetHandler.Create)
	protected.POST("/targets/bulk-create", targetHandler.BatchCreate)
	protected.POST("/targets/bulk-delete", targetHandler.BulkDelete)
	protected.GET("/targets", targetHandler.List)
	protected.GET("/targets/:id", targetHandler.GetByID)
	protected.PUT("/targets/:id", targetHandler.Update)
	protected.DELETE("/targets/:id", targetHandler.Delete)
}
