package router

import (
	"github.com/gin-gonic/gin"
	assethandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler"
)

func registerAssetStatisticsRoutes(protected *gin.RouterGroup, handler *assethandler.AssetStatisticsHandler) {
	protected.GET("/assetStatistics", handler.GetCurrent)
	protected.GET("/assetStatistics/history", handler.ListHistory)
}
