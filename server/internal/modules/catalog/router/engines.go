package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

func registerEngineCatalogRoutes(protected *gin.RouterGroup, engineCatalogHandler *handler.EngineCatalogHandler) {
	protected.GET("/engines", engineCatalogHandler.List)
	protected.GET("/engines/:engine", engineCatalogHandler.GetByID)
	protected.POST("/engines:installMethod", func(c *gin.Context) {
		httpdto.DispatchCustomMethod(c, "installMethod", map[string]gin.HandlerFunc{"install": engineCatalogHandler.Install})
	})
}
