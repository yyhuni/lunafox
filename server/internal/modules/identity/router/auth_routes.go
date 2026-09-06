package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/handler"
)

func registerAuthRoutes(api *gin.RouterGroup, authHandler *handler.AuthHandler) {
	api.POST("/sessions", authHandler.Login)
	api.POST("/sessions:renew", authHandler.RefreshToken)
}

func registerAuthProtectedRoutes(protected *gin.RouterGroup, authHandler *handler.AuthHandler) {
	protected.GET("/users/current", authHandler.GetCurrentUser)
}
