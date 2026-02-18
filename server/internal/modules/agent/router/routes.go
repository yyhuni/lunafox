package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	agenthandler "github.com/yyhuni/lunafox/server/internal/modules/agent/handler"
)

// RegisterAgentRoutes registers agent-facing routes.
func RegisterAgentRoutes(
	api *gin.RouterGroup,
	protected *gin.RouterGroup,
	agentHandler *agenthandler.AgentHandler,
	agentWSHandler *agenthandler.AgentWebSocketHandler,
	agentTaskHandler *agenthandler.AgentTaskHandler,
	agentRepo agentdomain.AgentRepository,
) {
	runtime := api.Group("/agent")
	runtime.POST("/register", agentHandler.Register)
	runtime.GET("/install-script", agentHandler.InstallScript)
	runtime.GET("/ws", middleware.AgentAuthMiddleware(agentRepo), agentWSHandler.Handle)

	taskAPI := runtime.Group("")
	taskAPI.Use(middleware.AgentAuthMiddleware(agentRepo))
	{
		taskAPI.POST("/tasks/pull", agentTaskHandler.PullTask)
		taskAPI.PATCH("/tasks/:taskId/status", agentTaskHandler.UpdateTaskStatus)
	}

	admin := protected.Group("/admin/agents")
	{
		admin.POST("/registration-tokens", agentHandler.CreateRegistrationToken)
		admin.GET("", agentHandler.ListAgents)
		admin.GET("/:id", agentHandler.GetAgent)
		admin.DELETE("/:id", agentHandler.DeleteAgent)
		admin.PATCH("/:id/config", agentHandler.UpdateAgentConfig)
	}
}
