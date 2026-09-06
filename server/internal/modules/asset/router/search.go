package router

import (
	"github.com/gin-gonic/gin"
	searchhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/search"
)

func registerGlobalAssetSearchRoutes(protected *gin.RouterGroup, handler *searchhandler.GlobalAssetSearchHandler) {
	protected.GET("/assets:search", handler.Search)
}
