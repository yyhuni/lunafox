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
	agentClusterSummaryHandler *agenthandler.AgentClusterSummaryHandler,
	agentLocationMapHandler *agenthandler.AgentLocationMapHandler,
) {
	api.POST("/agents:register", agentHandler.Register)
	api.GET("/agents:downloadInstallScript", agentHandler.DownloadInstallScript)
	protected.POST("/admin/agentRegistrationTokens", agentHandler.CreateRegistrationToken)
	protected.GET("/admin/agentRegistrationTokens/:registrationToken", agentHandler.GetRegistrationToken)

	admin := protected.Group("/admin/agents")
	{
		admin.GET("", agentHandler.List)
		admin.GET("/filterOptions", agentHandler.FilterOptions)
		admin.GET("/:agent", agentHandler.GetAgent)
		admin.DELETE("/:agent", agentHandler.DeleteAgent)
		admin.PATCH("/:agent", agentHandler.UpdateAgentConfig)
		admin.GET("/:agent/logEntries", agentLogHandler.List)
	}
	protected.GET("/admin/agentClusterSummaries/current", agentClusterSummaryHandler.Current)
	protected.GET("/admin/agentLocationMaps/current", agentLocationMapHandler.Current)
}
