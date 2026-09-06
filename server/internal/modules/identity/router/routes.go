package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/handler"
)

// RegisterIdentityRoutes registers auth/user/organization routes.
func RegisterIdentityRoutes(api *gin.RouterGroup, protected *gin.RouterGroup, authHandler *handler.AuthHandler, userHandler *handler.UserHandler, orgHandler *handler.OrganizationHandler) {
	registerAuthRoutes(api, authHandler)
	registerAuthProtectedRoutes(protected, authHandler)
	registerUserRoutes(protected, userHandler)
	registerOrganizationRoutes(protected, orgHandler)
}
