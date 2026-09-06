package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

func registerTargetRoutes(protected *gin.RouterGroup, targetHandler *handler.TargetHandler) {
	protected.POST("/targets", targetHandler.Create)
	protected.POST("/targets:batchMethod", func(c *gin.Context) {
		httpdto.DispatchCustomMethod(c, "batchMethod", map[string]gin.HandlerFunc{
			"batchCreate": targetHandler.BatchCreate,
			"batchDelete": targetHandler.BatchDelete,
		})
	})
	protected.GET("/targets", targetHandler.List)
	protected.GET("/targets/:target", targetHandler.GetByID)
	protected.PATCH("/targets/:target", targetHandler.Update)
	protected.DELETE("/targets/:target", targetHandler.Delete)
}
