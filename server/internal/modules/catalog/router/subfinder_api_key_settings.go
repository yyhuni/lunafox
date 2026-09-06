package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
)

func registerSubfinderAPIKeySettingsRoutes(protected *gin.RouterGroup, settingsHandler *handler.SubfinderAPIKeySettingsHandler) {
	protected.GET("/settings/apiKeys", settingsHandler.GetSettings)
	protected.PATCH("/settings/apiKeys", settingsHandler.UpdateSettings)
}
