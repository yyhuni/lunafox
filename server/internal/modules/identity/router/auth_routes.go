package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/handler"
)

func registerAuthRoutes(api *gin.RouterGroup, authHandler *handler.AuthHandler) {
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.RefreshToken)
	}
}

func registerAuthProtectedRoutes(protected *gin.RouterGroup, authHandler *handler.AuthHandler) {
	protected.GET("/auth/me", authHandler.GetCurrentUser)
}
