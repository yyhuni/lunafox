package router

import (
	"github.com/gin-gonic/gin"
	agenthandler "github.com/yyhuni/lunafox/server/internal/modules/agent/handler"
)

// RegisterAgentRoutes registers agent-facing routes.
func RegisterAgentRoutes(
	api *gin.RouterGroup,
	protected *gin.RouterGroup,
	agentHandler *agenthandler.AgentHandler,
	agentLogHandler *agenthandler.AgentLogHandler,
) {
	runtime := api.Group("/agent")
	runtime.POST("/register", agentHandler.Register)
	runtime.GET("/install-script", agentHandler.InstallScript)
	runtime.GET("/install-script/local", agentHandler.InstallScriptLocal)
	runtime.GET("/install-script/remote", agentHandler.InstallScriptRemote)

	admin := protected.Group("/admin/agents")
	{
		admin.POST("/registration-tokens", agentHandler.CreateRegistrationToken)
		admin.GET("", agentHandler.ListAgents)
		admin.GET("/:id", agentHandler.GetAgent)
		admin.DELETE("/:id", agentHandler.DeleteAgent)
		admin.PATCH("/:id/config", agentHandler.UpdateAgentConfig)
		admin.GET("/:id/logs", agentLogHandler.List)
	}
}
