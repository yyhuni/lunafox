package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/handler"
)

func registerUserRoutes(protected *gin.RouterGroup, userHandler *handler.UserHandler) {
	protected.POST("/users", userHandler.CreateUser)
	protected.GET("/users", userHandler.ListUsers)
	protected.PUT("/users/me/password", userHandler.UpdateCurrentUserPassword)
}
