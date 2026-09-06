package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/handler"
)

func registerUserRoutes(protected *gin.RouterGroup, userHandler *handler.UserHandler) {
	protected.POST("/users", userHandler.CreateUser)
	protected.GET("/users", userHandler.List)
	protected.POST("/users/me:customMethod", func(c *gin.Context) {
		httpdto.DispatchCustomMethod(c, "customMethod", map[string]gin.HandlerFunc{
			"changePassword": userHandler.UpdateCurrentUserPassword,
			"generateMcpKey": userHandler.GenerateCurrentMCPKey,
		})
	})
	protected.GET("/users/me/mcpKey", userHandler.GetCurrentMCPKey)
}
